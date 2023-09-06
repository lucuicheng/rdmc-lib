package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"rdmc/internal"
	"rdmc/internal/extract"
	"rdmc/pkg"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type ReportFRSItem struct {
	Score float64 `json:"score"`
	Type  string  `json:"type"`
}

type ReportFRSHealthChecks struct {
	Item       []ReportFRSItem `json:"item"`
	TotalScore float64         `json:"totalScore"`
}

type ReportFRSItemDetection struct {
	Type  string `json:"type,omitempty"`
	Count int    `json:"count"`
	Path  string `json:"path"`
}

type ReportFRS struct {
	CheckName               string                   `json:"checkName"`
	Ip                      string                   `json:"ip"`
	CheckPath               string                   `json:"checkPath"`
	FileCount               int                      `json:"fileCount"`
	HealthChecks            ReportFRSHealthChecks    `json:"healthChecks"`
	FileDetection           []ReportFRSItemDetection `json:"fileDetection"`
	RansomFile              []ReportFRSItemDetection `json:"ransomFile"`
	EigenvalueDetection     []ReportFRSItemDetection `json:"eigenvalueDetection"`
	SecretSecurityDetection []ReportFRSItemDetection `json:"secretSecurityDetection"`
	PermissionDetection     []ReportFRSItemDetection `json:"permissionDetection"`
	DownloadContent         string                   `json:"downloadContent"`
	//--
	SuspiciousList      []extract.Output `json:"suspiciousList"`
	OaList              []extract.Output `json:"oaList"`
	PdfList             []extract.Output `json:"pdfList"`
	ExtortionList       []extract.Output `json:"extortionList"`
	Permissions777List  []extract.Output `json:"permissions777List"`
	PermissionsSuidList []extract.Output `json:"permissionsSuidList"`
	PermissionsSgidList []extract.Output `json:"permissionsSgidList"`
	OaPwdList           []extract.Output `json:"oaPwdList"`
	PdfPwdList          []extract.Output `json:"pdfPwdList"`
	ZipPwdList          []extract.Output `json:"zipPwdList"`
}

func handleScanReader(reader *bufio.Reader) (int, error) {
	count := 0
	for {
		str, err := reader.ReadString('\n')
		if len(str) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return count, err
		}

		//log := logger.Appender().WithField("pid", pid).WithField("ppid", ppid)

		if strings.Contains(str, "Count:") {
			content := strings.Replace(str, "\n", "", -1)
			content = strings.Replace(content, "Count:", "", -1)
			count, _ = strconv.Atoi(content)
			//------------------
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return count, err
		}
	}

	return count, nil
}

func RunCommand(operation string, originTask Task, run func(task Task), done func(task Task), failed func(task Task, err error)) (int, error) {

	startDateTime := time.Now()

	task := Task{
		Name:          originTask.Name,
		Phase:         operation,
		Path:          originTask.Path,
		Count:         originTask.Count,
		Ids:           originTask.Ids,
		MainId:        originTask.MainId,
		StartDateTime: startDateTime,
		StartDate:     startDateTime.Format("2006-01-02 15:04:05"),
	}

	var args []string
	var count int

	// 优化参数组织形式
	args = append(args, "-"+operation)

	//args = append(args, "/opt/rdmc/lib/bait")
	args = append(args, "-root")
	args = append(args, task.Path)

	if task.Name != "" {
		args = append(args, "-task")
		args = append(args, task.Name)
	}

	if operation == "download" {
		if task.MainId != 0 {
			args = append(args, "-folder")
			args = append(args, strconv.Itoa(task.MainId))
		}

		if task.Ids != "" {
			args = append(args, "-id")
			args = append(args, task.Ids)
		}
	}

	args = append(args, "-count")
	args = append(args, strconv.Itoa(task.Count))

	//启动监控进程 nohup /opt/rdmc/lib/baits -scan -target /opt/baits  > output 2>&1 &
	command := exec.Command("/opt/rdmc/lib/extract", args...)
	//command := exec.Command("sleep", "3600")
	//command := exec.Command("/bin/sh", "-x", "/opt/bait.sh")
	log.Println(command.String())

	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed creating command stdoutpipe: %s", err)
		return count, err
	}
	defer stdout.Close()
	stdoutReader := bufio.NewReader(stdout)

	stderr, err := command.StderrPipe()
	if err != nil {
		log.Fatalf("Failed creating command stderrpipe: %s", err)
		return count, err
	}
	defer stderr.Close()
	stderrReader := bufio.NewReader(stderr)

	if err := command.Start(); err != nil {
		log.Fatalf("Failed run command: %s", err)
		return count, err
	}

	task.Pid = command.Process.Pid
	run(task)

	count, _ = handleScanReader(stdoutReader)
	internal.HandleErrorReader(stderrReader)

	if err := command.Wait(); err != nil {
		log.Printf("start scan %d exit with err: %v\n", command.Process.Pid, err)
		if err, ok := err.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Stop Error Exit Status: %d\n", status.ExitStatus())
				return count, err
			}
		}
		fmt.Printf("Run Error : %s \n", err)
		failed(task, err)
		return count, err
	}

	done(task)
	return count, nil
}

type Task struct {
	Name          string          `json:"name"`
	Token         string          `json:"token"`
	Pid           int             `json:"pid"`
	StartDate     string          `json:"startDate"`
	StartDateTime time.Time       `json:"startDateTime"`
	EndDate       string          `json:"endDate"`
	Duration      string          `json:"duration"`
	Host          string          `json:"host"`
	Path          string          `json:"path"`
	VirtualPath   string          `json:"virtualPath"`
	Status        string          `json:"status"`
	Process       map[string]Task `json:"process"`
	Phase         string          `json:"phase"`
	Count         int             `json:"count"`
	MainId        int             `json:"mainId"`
	Ids           string          `json:"ids"`
}

func (t *Task) Init(taskName string) {
	if taskName == "" {
		taskName = "Task_0000"
	}
	t.Name = taskName
	t.StartDate = t.StartDateTime.Format("2006-01-02 15:04:05")
}

func (t *Task) PhaseStatus(host string, token string, status string) []byte {

	t.Status = status
	t.Token = token // TODO 优化 token 的传输，其实不需要传输 token 内容
	t.Host = host

	taskInfoBytes, _ := json.Marshal(t)

	postParams := make(map[string]string)
	postParams["path"] = fmt.Sprintf(`[FRS]%s`, t.Name)
	postParams["pid"] = string(taskInfoBytes)

	paramsBytes, _ := json.Marshal(postParams)

	return paramsBytes
}

func (t *Task) PhaseRunning(message string, callbackHost string, token string, config internal.Config) {
	statusMessage := message

	//t.StartDateTime = time.Now()
	//t.StartDate = t.StartDateTime.Format("2006-01-02 15:04:05")

	paramsBytes := t.PhaseStatus(callbackHost, token, statusMessage)
	if len(config.Masters) > 0 {
		for _, master := range config.Masters {
			internal.Register(master.Address, master.Port, token, string(paramsBytes))
		}
	}
}

func (t *Task) PhaseFinished(message string, process map[string]Task, phase Task, callbackHost string, token string, config internal.Config) {
	statusMessage := message

	phase.EndDate = time.Now().Format("2006-01-02 15:04:05.000")
	phase.Duration = fmt.Sprintf("%v", time.Since(phase.StartDateTime).Round(time.Millisecond))

	process[phase.Phase] = phase //记录 流程信息
	t.Process = process          // 追加运行记录

	paramsBytes := t.PhaseStatus(callbackHost, token, statusMessage)
	if len(config.Masters) > 0 {
		for _, master := range config.Masters {
			internal.Register(master.Address, master.Port, token, string(paramsBytes))
		}
	}
}

func (t *Task) ProcessFinished(message string, process map[string]Task, callbackHost string, token string, config internal.Config) {
	statusMessage := message
	t.Process = process // 追加运行记录

	t.EndDate = time.Now().Format("2006-01-02 15:04:05")
	t.Duration = fmt.Sprintf("%v", time.Since(t.StartDateTime).Round(time.Millisecond))

	paramsBytes := t.PhaseStatus(callbackHost, token, statusMessage)
	if len(config.Masters) > 0 {
		for _, master := range config.Masters {
			internal.Register(master.Address, master.Port, token, string(paramsBytes))
		}
	}
}

const (
	reportTemplate = ``
)

func main() {

	//
	//reinstall := flag.Bool("reinstall", true, "")
	//
	scan := flag.Bool("scan", false, "")
	report := flag.Bool("report", false, "")

	netdisk := flag.Bool("netdisk", false, "")
	//
	//uninstall := flag.Bool("uninstall", true, "")

	version := flag.Bool("v", false, "Prints current tool version")
	daemon := flag.Bool("d", false, "run command in daemon")
	token := flag.String("token", "", "run command token")

	//Define the root directory to scan
	//sourcePath := flag.String("source", "tmp/123456789-deduplication-1684145230872.txt", "The md5 check source file path")
	targetPath := flag.String("target", "", "the scan path")
	ids := flag.String("id", "", "file need sync id(s)")
	folderId := flag.Int("folder", 1, "root folder id")

	task := flag.String("task", "Task_0000", "the task name")
	host := flag.String("host", "localhost", "the task running host")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - FRS Scan v%s", pkg.AppVersion))
		os.Exit(0)
		return
	}

	// 运行多步骤扫描任务
	if *scan && !*daemon {

		frsScanPath := *targetPath
		taskName := *task
		callbackHost := *host

		startTime := time.Now()

		taskInfo := &Task{
			Pid:           os.Getpid(),
			Name:          taskName,
			StartDateTime: time.Now(),
			Path:          frsScanPath,
			Count:         0,
			MainId:        *folderId,
			Ids:           *ids,
		}
		taskInfo.Init(taskInfo.Name) // 初始化默认 task， TODO 优化顺序

		config, _ := internal.ReadConfig()

		// 路径存在检测, 只有非网盘才需要检测路径是否存在
		if !*netdisk {
			if !internal.IsExist(frsScanPath) {

				taskInfo.EndDate = time.Now().Format("2006-01-02 15:04:05")
				taskInfo.Duration = fmt.Sprintf("%v", time.Since(taskInfo.StartDateTime).Round(time.Millisecond))

				paramsBytes := taskInfo.PhaseStatus(callbackHost, *token, "failed: path not exist")
				if len(config.Masters) > 0 {
					for _, master := range config.Masters {
						internal.Register(master.Address, master.Port, *token, string(paramsBytes))
					}
				}

				return
			}
		}

		// 清理重复作业目录
		os.RemoveAll(fmt.Sprintf("/opt/frs/%s", taskInfo.Name))
		//keep Running := true

		// 需要执行多个命令 组成完整查询
		log.Printf("running task %s pid is [%d]\n", taskInfo.Name, os.Getpid())
		taskInfo.PhaseRunning("running", callbackHost, *token, config)

		var stepTime time.Time
		process := map[string]Task{}

		if *netdisk {

			taskInfo.VirtualPath = fmt.Sprintf("%s", taskInfo.Path) // 显示用的虚拟路径

			stepTime = time.Now()
			// 扫描，需要增加额外的参数
			fetchedCount, _ := RunCommand("fetch", *taskInfo,
				// TODO 注册 startup 与 存活列表,发送状态信息，发送失败，提示未同步成功，本地服务未开启，原则上不会出现该问题，持久到本地数据库
				func(task Task) {
					// 开始阶段
					taskInfo.PhaseRunning("fetch running", callbackHost, *token, config)
				},
				func(task Task) {
					// 运行成功
					taskInfo.PhaseFinished("fetch successfully", process, task, callbackHost, *token, config)
				},
				func(task Task, err error) {
					// 运行失败
					taskInfo.PhaseFinished(fmt.Sprintf("fetch failed: %v", err), process, task, callbackHost, *token, config)
				},
			)
			log.Printf("Finish Extract Task [Fetch] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), fetchedCount)
			taskInfo.Count = fetchedCount

			stepTime = time.Now()
			// 同步文件 需要增加额外的参数
			downloadedCount, _ := RunCommand("download", *taskInfo,
				func(task Task) {
					// 开始阶段
					taskInfo.PhaseRunning("download running", callbackHost, *token, config)
				},
				func(task Task) {
					// 运行成功
					taskInfo.PhaseFinished("download successfully", process, task, callbackHost, *token, config)
				},
				func(task Task, err error) {
					// 运行失败
					taskInfo.PhaseFinished(fmt.Sprintf("download failed: %v", err), process, task, callbackHost, *token, config)
				},
			)
			log.Printf("Finish Extract Task [Download] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), downloadedCount)
			taskInfo.Count = downloadedCount

			// 特殊的路径封装
			prefix := fmt.Sprintf("/opt/frs/%s", taskName)
			taskInfo.Path = fmt.Sprintf("%s/%s", prefix, taskInfo.Path) // 后续步骤的实际路径

		} else {
			originCount, _ := RunCommand("origin", *taskInfo,
				func(task Task) {
					// 开始阶段
					taskInfo.PhaseRunning("origin running", callbackHost, *token, config)
				},
				func(task Task) {
					// 运行成功
					taskInfo.PhaseFinished("origin successfully", process, task, callbackHost, *token, config)
				},
				func(task Task, err error) {
					// 运行失败
					taskInfo.PhaseFinished(fmt.Sprintf("origin failed: %v", err), process, task, callbackHost, *token, config)
				},
			)
			log.Printf("Finish Extract Task [Origin] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), originCount)
			taskInfo.Count = originCount
		}

		// 顺序执行 具体检测任务 TODO 改成并发，采集具体执行错误信息

		stepTime = time.Now()
		count, _ := RunCommand("duplicate", *taskInfo,
			func(task Task) {
				// 开始阶段
				taskInfo.PhaseRunning("duplicate running", callbackHost, *token, config)
			},
			func(task Task) {
				// 运行成功
				taskInfo.PhaseFinished("duplicate successfully", process, task, callbackHost, *token, config)
			},
			func(task Task, err error) {
				// 运行失败
				taskInfo.PhaseFinished(fmt.Sprintf("duplicate failed: %v", err), process, task, callbackHost, *token, config)
			},
		)
		log.Printf("Finish Extract Task [Duplicate] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), count)

		stepTime = time.Now()
		count, _ = RunCommand("permissions", *taskInfo,
			func(task Task) {
				// 开始阶段
				taskInfo.PhaseRunning("permissions running", callbackHost, *token, config)
			},
			func(task Task) {
				// 运行成功
				taskInfo.PhaseFinished("permissions successfully", process, task, callbackHost, *token, config)
			},
			func(task Task, err error) {
				// 运行失败
				taskInfo.PhaseFinished(fmt.Sprintf("permissions failed: %v", err), process, task, callbackHost, *token, config)
			},
		)
		log.Printf("Finish Extract Task [Permissions] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), count)

		stepTime = time.Now()
		count, _ = RunCommand("suffix", *taskInfo,
			func(task Task) {
				// 开始阶段
				taskInfo.PhaseRunning("suffix running", callbackHost, *token, config)
			},
			func(task Task) {
				// 运行成功
				taskInfo.PhaseFinished("suffix successfully", process, task, callbackHost, *token, config)
			},
			func(task Task, err error) {
				// 运行失败
				taskInfo.PhaseFinished(fmt.Sprintf("suffix failed: %v", err), process, task, callbackHost, *token, config)
			},
		)
		log.Printf("Finish Extract Task [Suffix] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), count)

		stepTime = time.Now()
		count, _ = RunCommand("content", *taskInfo,
			func(task Task) {
				// 开始阶段
				taskInfo.PhaseRunning("content running", callbackHost, *token, config)
			},
			func(task Task) {
				// 运行成功
				taskInfo.PhaseFinished("content successfully", process, task, callbackHost, *token, config)
			},
			func(task Task, err error) {
				// 运行失败
				taskInfo.PhaseFinished(fmt.Sprintf("content failed: %v", err), process, task, callbackHost, *token, config)
			},
		)
		log.Printf("Finish Extract Task [Content] Cost=[%v], Files Count=[%d] \n", time.Since(stepTime), count)

		// 最后运行完成，计算时间
		taskInfo.ProcessFinished("successfully", process, callbackHost, *token, config)

		// 最后计算 运行持续时间
		//fmt.Println(count, startTime)
		log.Printf("Finish Extract Task [Full] Cost=[%v], Files Count=[%d] \n", time.Since(startTime), count)

		return
	}

	// 运行多步骤扫描任务 后台运行方式
	if *scan && *daemon {
		if *netdisk {
			command := exec.Command("/opt/rdmc/lib/frs",
				"-scan", "-host", *host, "-target", *targetPath, "-folder", strconv.Itoa(*folderId), "-id", *ids, "-task", *task, "-netdisk", "-token", *token)
			log.Println(*host, command.String())

			if err := command.Start(); err != nil {
				fmt.Printf("start in background commmand error : %v\n", err)
			}
			os.Exit(0)
		} else {
			command := exec.Command("/opt/rdmc/lib/frs", "-scan", "-host", *host, "-target", *targetPath, "-task", *task, "-token", *token)
			log.Println(command.String())

			if err := command.Start(); err != nil {
				fmt.Printf("start in background commmand error : %v\n", err)
			}
			os.Exit(0)
		}

		return
	}

	// 运行报告生成解析实时数据
	if *report {
		file, err := os.Open(fmt.Sprintf("/opt/frs/%s/report.frs", *task))
		if err != nil {
			log.Fatal("Failed to open file_paths.txt:", err)
		}
		defer file.Close()

		reportFRS := ReportFRS{
			FileCount: 0,
			HealthChecks: ReportFRSHealthChecks{
				[]ReportFRSItem{
					{0.0, "FILE_DETECTION"},
					{0.0, "EIGENVALUE_DETECTION"},
					{0.0, "RANSOM_FILE"},
					{0.0, "PERMISSION_DETECTION"},
					{0.0, "SECRET_SECURITY_DETECTION"},
				},
				0.0,
			},
			FileDetection:           []ReportFRSItemDetection{{"", 0, ""}},
			RansomFile:              []ReportFRSItemDetection{{"", 0, ""}},
			EigenvalueDetection:     []ReportFRSItemDetection{{"", 0, ""}},
			SecretSecurityDetection: []ReportFRSItemDetection{{"", 0, ""}},
			PermissionDetection:     []ReportFRSItemDetection{{"", 0, ""}},
			//--
			SuspiciousList:      []extract.Output{},
			OaList:              []extract.Output{},
			PdfList:             []extract.Output{},
			ExtortionList:       []extract.Output{},
			Permissions777List:  []extract.Output{},
			PermissionsSuidList: []extract.Output{},
			PermissionsSgidList: []extract.Output{},
			OaPwdList:           []extract.Output{},
			PdfPwdList:          []extract.Output{},
			ZipPwdList:          []extract.Output{},
		}

		var wg sync.WaitGroup                         // 创建一个等待组
		maxConcurrency := 5                           // 最大并发打开文件数
		lineChan := make(chan string, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

		var fileDetectionCount int64
		var fileDetection []ReportFRSItemDetection
		var ransomFileDetectionCount int64
		var ransomFileDetection []ReportFRSItemDetection

		var eigenvalueDetectionOfficeCount int64
		var eigenvalueDetectionPDFCount int64

		var secretSecurityDetectionOfficeCount int64
		var secretSecurityDetectionPDFCount int64
		var secretSecurityDetectionCompressedPackageCount int64

		var permissionDetection777Count int64
		var permissionDetectionSUIDCount int64
		var permissionDetectionSGIDCount int64

		// 启动多个 goroutine 处理文件
		for i := 0; i < maxConcurrency-1; i++ {
			wg.Add(1)
			go func() {
				for line := range lineChan {
					// 解析 行数据
					var detail []extract.Output
					if len(strings.Split(line, ", Detail: ")) > 1 {
						json.Unmarshal([]byte(strings.Split(line, ", Detail: ")[1]), &detail)
					}
					line := strings.Split(line, ", Detail: ")[0]

					count := 0
					parts := strings.Split(line, ": ")
					lastPart := parts[len(parts)-1]
					if strings.Contains(lastPart, "(") { // 理论上始终会有 至少 2个部分组成
						count, _ = strconv.Atoi(strings.Split(lastPart, "(")[0])
					} else {
						count, _ = strconv.Atoi(lastPart)
					}
					name := parts[1]
					if strings.Contains(name, "(") {
						name = strings.Split(name, "(")[0]
					}

					// 总数
					if strings.Contains(line, "FileCount: ") {
						reportFRS.FileCount = count
						s := strings.Split(lastPart, "(")[1]
						reportFRS.CheckPath = s[:len(s)-1]
					}

					// 基础类型
					if strings.Contains(line, "DUPLICATED: ") {
						// 勒索文件检测
						atomic.AddInt64(&ransomFileDetectionCount, int64(count))
						ransomFileDetection = append(ransomFileDetection, ReportFRSItemDetection{Path: name, Count: count})
						// list
						reportFRS.ExtortionList = append(reportFRS.ExtortionList, detail...)
					}
					if strings.Contains(line, "PERMISSION: ") {
						// 权限检测
						if strings.Contains(line, "777=true") {
							atomic.AddInt64(&permissionDetection777Count, int64(count))
							// list
							reportFRS.Permissions777List = append(reportFRS.Permissions777List, detail...)
						}
						if strings.Contains(line, "suid=true") {
							atomic.AddInt64(&permissionDetectionSUIDCount, int64(count))
							// list
							reportFRS.PermissionsSuidList = append(reportFRS.PermissionsSuidList, detail...)
						}
						if strings.Contains(line, "sgid=true") {
							atomic.AddInt64(&permissionDetectionSGIDCount, int64(count))
							// list
							reportFRS.PermissionsSgidList = append(reportFRS.PermissionsSgidList, detail...)
						}
					}
					if strings.Contains(line, "SUFFIX: ") {
						// 可疑文件检测
						atomic.AddInt64(&fileDetectionCount, int64(count))
						fileDetection = append(fileDetection, ReportFRSItemDetection{Path: name, Count: count})
						// list
						reportFRS.SuspiciousList = append(reportFRS.SuspiciousList, detail...)
					}
					if strings.Contains(line, "CONTENT: ") {
						// 密保检测
						if strings.Contains(line, "true") {
							if strings.Contains(line, ".docx") || strings.Contains(line, ".xlsx") || strings.Contains(line, ".pptx") { // OA
								atomic.AddInt64(&secretSecurityDetectionOfficeCount, int64(count))
								//list
								reportFRS.OaPwdList = append(reportFRS.OaPwdList, detail...)
							} else if strings.Contains(line, ".PDF") || strings.Contains(line, ".pdf") { // pdf
								atomic.AddInt64(&secretSecurityDetectionPDFCount, int64(count))
								//list
								reportFRS.PdfPwdList = append(reportFRS.PdfPwdList, detail...)
							} else if strings.Contains(line, ".zip") || strings.Contains(line, ".tar.gz") { //compressed package
								atomic.AddInt64(&secretSecurityDetectionCompressedPackageCount, int64(count))
								//list
								reportFRS.ZipPwdList = append(reportFRS.ZipPwdList, detail...)
							}
						}
						// 特征值检测
						if strings.Contains(line, "false") && strings.Contains(line, "unmatched-suffix-") {
							if strings.Contains(line, ".docx") || strings.Contains(line, ".xlsx") || strings.Contains(line, ".pptx") { // OA
								atomic.AddInt64(&eigenvalueDetectionOfficeCount, int64(count))
								//list
								reportFRS.OaList = append(reportFRS.OaList, detail...)
							} else if strings.Contains(line, ".PDF") || strings.Contains(line, ".pdf") { // pdf
								atomic.AddInt64(&eigenvalueDetectionPDFCount, int64(count))
								//list
								reportFRS.PdfList = append(reportFRS.PdfList, detail...)
							}
						}
					}
					//fmt.Println(line)
				}
				wg.Done()
			}()
		}

		go func() {
			// 启动多个 goroutine 来计算文件的 MD5 值
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				lineChan <- line
			}
			close(lineChan) // 关闭通道，告知处理文件的 goroutine 没有更多任务
		}()

		wg.Wait()

		atomic.LoadInt64(&ransomFileDetectionCount)
		atomic.LoadInt64(&fileDetectionCount)

		reportFRS.FileDetection = fileDetection    //[]ReportFRSItemDetection{{"", int(atomic.LoadInt64(&fileDetectionCount)), ""}}    // 可疑文件检测
		reportFRS.RansomFile = ransomFileDetection //[]ReportFRSItemDetection{{"", int(atomic.LoadInt64(&ransomFileDetectionCount)), ""}} // 勒索文件检测

		reportFRS.EigenvalueDetection = []ReportFRSItemDetection{ // 特征值文件检测
			{"OA", int(atomic.LoadInt64(&eigenvalueDetectionOfficeCount)), ""},
			{"PDF", int(atomic.LoadInt64(&eigenvalueDetectionPDFCount)), ""},
		}
		reportFRS.SecretSecurityDetection = []ReportFRSItemDetection{ //密码保护文件检测
			{"OA", int(atomic.LoadInt64(&secretSecurityDetectionOfficeCount)), ""},
			{"PDF", int(atomic.LoadInt64(&secretSecurityDetectionPDFCount)), ""},
			{"compressed package", int(atomic.LoadInt64(&secretSecurityDetectionCompressedPackageCount)), ""},
		}
		reportFRS.PermissionDetection = []ReportFRSItemDetection{
			{"777", int(atomic.LoadInt64(&permissionDetection777Count)), ""},
			{"SUID", int(atomic.LoadInt64(&permissionDetectionSUIDCount)), ""},
			{"SGID", int(atomic.LoadInt64(&permissionDetectionSGIDCount)), ""},
		} // 权限检测

		// 计算分数
		wrapperLessZero := func(score float64) float64 {
			if score < 0.0 {
				return 0.0
			} else {
				return score
			}
		}

		designScore := 100.0
		if eigenvalueDetectionOfficeCount+eigenvalueDetectionPDFCount > 0 {
			designScore = 90.0
		}

		fileDetectionScore := wrapperLessZero(100.0 - math.Ceil(float64(fileDetectionCount)/float64(reportFRS.FileCount)))
		eigenvalueDetectionScore := wrapperLessZero(designScore - math.Ceil(float64(eigenvalueDetectionOfficeCount+eigenvalueDetectionPDFCount)))
		ransomDetectionScore := wrapperLessZero(100.0 - math.Ceil(float64(ransomFileDetectionCount)/10.0))
		permissionDetectionScore := wrapperLessZero(100.0 - math.Ceil(float64(permissionDetection777Count+permissionDetectionSUIDCount+permissionDetectionSGIDCount)/100.0))
		secretSecurityDetectionScore := wrapperLessZero(100.0 - math.Ceil(float64(secretSecurityDetectionOfficeCount+secretSecurityDetectionPDFCount+secretSecurityDetectionCompressedPackageCount)/10))

		checkedScore := ReportFRSHealthChecks{
			[]ReportFRSItem{
				{fileDetectionScore, "FILE_DETECTION"},                      //可疑文件检测 分数
				{eigenvalueDetectionScore, "EIGENVALUE_DETECTION"},          // 特征值文件检测 分数
				{ransomDetectionScore, "RANSOM_FILE"},                       //勒索文件检测 分数
				{permissionDetectionScore, "PERMISSION_DETECTION"},          // 权限检测 分数
				{secretSecurityDetectionScore, "SECRET_SECURITY_DETECTION"}, //密码保护文件检测 分数
			},
			fileDetectionScore*0.5 + eigenvalueDetectionScore*0.2 + ransomDetectionScore*0.1 + permissionDetectionScore*0.1 + secretSecurityDetectionScore*0.1,
		}
		reportFRS.HealthChecks = checkedScore

		// 生成下载内容
		tmpl := template.Must(template.New("example").Parse(reportTemplate)) // serverYMLTemplate
		var tmplBytes bytes.Buffer

		if err := tmpl.Execute(&tmplBytes, reportFRS); err != nil {
			log.Fatalf("Failed to execute template.\nError: %s", err.Error())
		}
		reportFRS.DownloadContent = base64.StdEncoding.EncodeToString(tmplBytes.Bytes())

		// 输出内容
		log.Println("report data: ")
		bytes, _ := json.Marshal(reportFRS)
		fmt.Println(string(bytes))
	}

}
