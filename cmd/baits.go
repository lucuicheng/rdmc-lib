package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"rdmc/internal"
	"rdmc/pkg"
	"strconv"
	"syscall"
	"time"
)

var (
	FILES = []string{
		"___aaa-password-rec",
		"_Global Supplier List.doc",
		"_IRM records.pdf",
		"_Payment information list.doc",
	}
)

func main() {

	//
	install := flag.Bool("install", false, "")
	uninstall := flag.Bool("uninstall", false, "")
	//reinstall := flag.Bool("reinstall", true, "")
	//
	watch := flag.Bool("watch", false, "")
	watchList := flag.Bool("watched-list", false, "")
	unwatch := flag.Bool("unwatch", false, "")
	//
	//uninstall := flag.Bool("uninstall", true, "")

	version := flag.Bool("v", false, "Prints current tool version")
	force := flag.Bool("f", false, "force unwatch,erase the startup data")
	daemon := flag.Bool("d", false, "run command in daemon")

	//Define the root directory to scan
	//sourcePath := flag.String("source", "tmp/123456789-deduplication-1684145230872.txt", "The md5 check source file path")
	targetPath := flag.String("target", "", "The md5 filteredCount file output path")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - Bait v%s", pkg.AppVersion))
		os.Exit(0)
		return
	}

	if *install {

		if internal.IsExist(*targetPath) {

			// TODO 检测是否是 nfs 目录
			if false {
				fmt.Printf("[%s] is mounted by nfs\n", *targetPath)
				return
			}

			for _, fileName := range FILES {
				internal.CreatBait(fileName, *targetPath)
			}
		} else {
			fmt.Printf("no such dir [%s]\n", *targetPath)
			return
		}

		/*var wg sync.WaitGroup // 创建一个等待组

		for _, fileName := range FILES {
			wg.Add(1)
			creatBait(fileName, *targetPath, &wg)
		}

		wg.Wait()*/
		return
	}

	if *watch && !*daemon {

		baitsWatchPath := *targetPath

		// watch 之前如果存在同 baitsWatchPath 的进程，杀死进程

		// 检测文件是否存在，不存在不执行
		var fullBaitsPaths []string
		var args []string
		baitsExisted := true

		for _, fileName := range FILES {
			fullBaitPath := path.Join(baitsWatchPath, fileName)
			baitExisted, _ := internal.BaitExists(fullBaitPath)
			if baitExisted {
				fullBaitsPaths = append(fullBaitsPaths, fullBaitPath)
			} else {
				baitsExisted = false
			}
		}

		if !baitsExisted {
			fmt.Println("not enough baits, expect 4 files")
			return
		}

		//args = append(args, "/opt/rdmc/lib/bait")
		args = append(args, "-m")
		args = append(args, fullBaitsPaths...)

		//启动监控进程 nohup /opt/rdmc/lib/baits -watch -target /opt/baits  > output 2>&1 &
		command := exec.Command("/opt/rdmc/lib/bait", args...)
		//command := exec.Command("sleep", "3600")
		//command := exec.Command("/bin/sh", "-x", "/opt/bait.sh")
		log.Println(command.String())

		stdout, err := command.StdoutPipe()
		if err != nil {
			log.Fatalf("Failed creating command stdoutpipe: %s", err)
			return
		}
		defer stdout.Close()
		stdoutReader := bufio.NewReader(stdout)

		stderr, err := command.StderrPipe()
		if err != nil {
			log.Fatalf("Failed creating command stderrpipe: %s", err)
			return
		}
		defer stderr.Close()
		stderrReader := bufio.NewReader(stderr)

		if err := command.Start(); err != nil {
			log.Println(err)
			return
		}

		// TODO 注册 startup 与 存活列表,发送状态信息，发送失败，提示未同步成功，本地服务未开启，原则上不会出现该问题，持久到本地数据库
		postParams := make(map[string]string)
		postParams["path"] = *targetPath
		postParams["pid"] = fmt.Sprintf("%d", command.Process.Pid)
		paramsBytes, _ := json.Marshal(postParams)

		log.Println("pid is", string(fmt.Sprintf("%d", command.Process.Pid)))

		internal.Register("localhost", 56790, "", string(paramsBytes))
		// 完整结束指定进程

		internal.HandleReader(stdoutReader)
		internal.HandleErrorReader(stderrReader)

		if err := command.Wait(); err != nil {
			log.Printf("start watch %d exit with err: %v\n", command.Process.Pid, err)
			if err, ok := err.(*exec.ExitError); ok {
				if status, ok := err.Sys().(syscall.WaitStatus); ok {
					fmt.Printf("Stop Error Exit Status: %d\n", status.ExitStatus())
				}
			}
			fmt.Printf("Run Error : %s \n", err)
			return
		}
		return
	}

	if *watch && *daemon {
		command := exec.Command("/opt/rdmc/lib/baits", "-watch", "-target", *targetPath)

		if err := command.Start(); err != nil {
			fmt.Printf("start in background commmand error : %v\n", err)
		}
		os.Exit(0)
		return
	}

	if *unwatch {

		result := internal.RunSubProcessDetect(*targetPath, false)

		if len(result) <= 1 && !*force {
			fmt.Println("no watching process at ", *targetPath)
			return
		}

		pid, _ := strconv.Atoi(result[0])
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			log.Printf("bait watch process killd error : %v", err)
			return
		}

		result = internal.RunProcessDetect(*targetPath, false)

		if len(result) <= 1 && !*force {
			fmt.Println("no wrapper watching process at ", *targetPath)
			return
		}

		pid, _ = strconv.Atoi(result[0])
		if err := syscall.Kill(pid, syscall.SIGKILL); err != nil {
			log.Printf("bait watch wrapper process killd error : %v", err)
			return
		}

		// TODO 进行取消 startup 设置
		deleteParams := make(map[string]string)
		deleteParams["path"] = *targetPath
		paramsBytes, _ := json.Marshal(deleteParams)
		internal.Erase("localhost", 56790, string(paramsBytes))

		os.Exit(0)
		return
	}

	if *watchList && !*daemon {
		// 列出所有监控信息

		result := internal.RunProcessWatching(false)

		if len(result) >= 3 {
			pid, _ := strconv.Atoi(result[0])
			fmt.Printf("already start another [%d] to monitor\n", pid)
			os.Exit(0)
			return
		}

		go func() {
			for {

				response := internal.List("localhost", 56790, "")

				if len(response.Body.Items) <= 0 {
					fmt.Println("no watched baits")
					return
				}

				var results []string

				for _, item := range response.Body.Items {
					for key, value := range item {
						status := 0
						msg := ""
						result := internal.RunProcessDetect(key, false)
						if len(result) <= 1 {
							msg = fmt.Sprintf("watched dir is [%s] and not running\n", key)
							status = 1 // 异常
						} else {
							msg = fmt.Sprintf("watched dir is [%s] and running pid is [%d]\n", key, value)
							status = 0 // 正常
						}

						// TODO 减少发送频率
						results = append(results, msg)
						// 状态同步
						params := map[string]interface{}{"directory": key, "liveStatus": status}
						paramsBytes, _ := json.Marshal(params)
						internal.SendStatus("localhost", 56790, string(paramsBytes))
					}
				}

				// 清空终端
				fmt.Print("\033[2J")
				// 将光标定位到第一行
				fmt.Print("\033[H")

				for i, result := range results {
					fmt.Printf("%d. %s", i+1, result)
				}

				now := time.Now().Format("2006-01-02 15:04:05")
				fmt.Printf("Current Time: %s\n", now)

				time.Sleep(10 * time.Second) // 改成 10s 进行一次数据同步
			}
		}()

		// 模拟其他操作，让程序保持运行
		select {}

		//command := exec.Command("/bin/sh", "-c", "ps -ef | grep -v grep | grep 'baits -watch ' | awk '{print $2,$11}' ")
		//log.Println(command.String())
		//
		//stdout, err := command.StdoutPipe()
		//if err != nil {
		//	log.Fatalf("Failed creating command stdoutpipe: %s", err)
		//	return
		//}
		//defer stdout.Close()
		//stdoutReader := bufio.NewReader(stdout)
		//
		//stderr, err := command.StderrPipe()
		//if err != nil {
		//	log.Fatalf("Failed creating command stderrpipe: %s", err)
		//	return
		//}
		//defer stderr.Close()
		//stderrReader := bufio.NewReader(stderr)
		//
		//if err := command.Start(); err != nil {
		//	log.Println(err)
		//	return
		//}
		//
		//handleProcessReader(stdoutReader)
		//handleErrorReader(stderrReader)
		//
		//// 完整结束指定进程
		//if err := command.Wait(); err != nil {
		//	log.Printf("Child command %d exit with err: %v\n", command.Process.Pid, err)
		//	return
		//}
	}

	if *watchList && *daemon {
		command := exec.Command("/opt/rdmc/lib/baits", "-watched-list")
		log.Println(command.String())

		if err := command.Start(); err != nil {
			fmt.Printf("start in background commmand error : %v\n", err)
		}
		os.Exit(0)
		return
	}

	if *uninstall {

		baitsWatchPath := *targetPath

		// 存在才能删除文件
		for _, fileName := range FILES {
			fullBaitPath := path.Join(baitsWatchPath, fileName)
			baitExisted, _ := internal.BaitExists(fullBaitPath)
			if baitExisted {
				os.Remove(fullBaitPath)
			}
		}

		os.Exit(0)
		return
	}
}
