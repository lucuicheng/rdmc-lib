package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"
)

type VerifiedDatabase struct {
	Name   string     `json:"name"`
	Column []string   `json:"column"`
	Data   [][]string `json:"data"`
	Warn   string     `json:"warn,omitempty"`
}

func signerFromPem(pemBytes []byte, password []byte) (ssh.Signer, error) {

	// read pem block
	err := errors.New("Pem decode failed, no key found")
	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, err
	}

	// handle encrypted key
	if x509.IsEncryptedPEMBlock(pemBlock) {

		fmt.Println(pemBlock.Type)

		// decrypt PEM
		pemBlock.Bytes, err = x509.DecryptPEMBlock(pemBlock, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("Decrypting PEM block failed %v", err)
		}

		// get RSA, EC or DSA key
		key, err := parsePemBlock(pemBlock)
		if err != nil {
			return nil, err
		}

		// generate signer instance from key
		signer, err := ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, fmt.Errorf("Creating signer from encrypted key failed %v", err)
		}

		return signer, nil
	} else {
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing plain private key failed %v", err)
		}

		return signer, nil
	}
}

func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
		} else {
			return key, nil
		}
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing EC private key failed %v", err)
		} else {
			return key, nil
		}
	case "DSA PRIVATE KEY":
		key, err := ssh.ParseDSAPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
		} else {
			return key, nil
		}
	default:
		return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", block.Type)
	}
}

func publicKeyAuthFunc(kPath string) ssh.AuthMethod {

	pemBytes, err := ioutil.ReadFile(kPath)
	if err != nil {
		log.Fatalf("Reading private key file failed %v\n", err)
	}
	// create signer
	signer, err := signerFromPem(pemBytes, []byte("P@ssw0rd"))
	if err != nil {
		log.Fatal("ssh key signer failed", err)
	}

	//keyPath, err := homedir.Expand(kPath)
	//if err != nil {
	//	log.Fatal("find key's home dir failed", err)
	//}
	//
	//key, err := ioutil.ReadFile(keyPath)
	//if err != nil {
	//	log.Fatal("ssh key file read failed", err)
	//}
	//
	//signer, err := ssh.ParsePrivateKey(key)
	//if err != nil {
	//	log.Fatal("ssh key signer failed", err)
	//}

	return ssh.PublicKeys(signer)
}

func readBuffForString(reader io.Reader, expect string) string {

	//re := regexp.MustCompile(`.*#$`)
	re := regexp.MustCompile(fmt.Sprintf(`.*%s$`, expect))

	excludeRE := regexp.MustCompile(fmt.Sprintf(`.*%s$`, "SQL>"))

	pattern := `(\x1B[\[(][0-?]*[ -/]*[@-~])` // 匹配 ANSI 代码的正则表达式

	// 使用正则表达式替换 ANSI 代码为空字符串
	reg := regexp.MustCompile(pattern)

	buf := make([]byte, 1000)
	n, err := reader.Read(buf) //this reads the ssh terminal
	waitingString := ""
	if err == nil {
		//for _, v := range buf[:n] {
		//	fmt.Printf("%c", v)
		//}
		waitingString = string(buf[:n])
	}

	for err == nil {
		// this loop will not end!!
		n, err = reader.Read(buf)
		waitingString += string(buf[:n])
		//for _, v := range buf[:n] {
		//	fmt.Printf("%c", v)
		//}

		cleanText := strings.TrimSpace(reg.ReplaceAllString(string(buf[:n]), ""))

		//strconv.Quote
		//fmt.Printf("---------------|%v|%v\n", strings.TrimSpace(cleanText), re.MatchString(strings.TrimSpace(cleanText)))
		if re.MatchString(cleanText) && !excludeRE.MatchString(cleanText) {
			break
		}

		if err != nil {
			//fmt.Println("------------", err)
		}
	}

	return waitingString
}

func commandExpect(continued chan bool, stdout io.Reader, expect string, handle func(content string)) {
	//result := make(chan string)
	//fmt.Printf(readBuffForString(stdout))
	go func(stdout io.Reader) {
		content := readBuffForString(stdout, expect)
		handle(content)
		//result <- content
		continued <- true
	}(stdout)

	<-continued
	//<-result
	//as := <-result
	//return as
}

func getPrevCommandResult(result []string) string {
	return result[len(result)-1]
}

func setLatestCommandResult(result []string, output string) []string {
	return append(result, output)
}

// GbkToUtf8 GBK 转 UTF-8
func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

// Utf8ToGbk UTF-8 转 GBK
func Utf8ToGbk(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
	d, e := ioutil.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}

func runWindowsCommand(session *ssh.Session, stopped chan bool, continued chan bool) []string {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("xterm", 400, 512, modes); err != nil {
		err = fmt.Errorf(
			"failed to request for pseudo terminal. servername: %s, err: %s",
			"dds", err)
		return nil
	}

	// StdinPipe for commands
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	//Start remote shell
	err = session.Shell()
	if err != nil {
		log.Fatal(err)
	}

	// 这段必须先执行，包装先开始异步阻塞
	go func() {
		for {
			select {
			case <-stopped:
				//fmt.Println("收到停止结束信号")

				stdin.Write([]byte("exit" + "\n")) // TODO 需要处理异常
				commandExpect(continued, stdout, ">", func(content string) {
					//fmt.Println(content)
				})

				//fmt.Println("收到手动释放/退出信号")
			default:
				// 没有阻塞信号时执行的操作
				//fmt.Println("执行其他操作")
				//time.Sleep(time.Second)
			}
		}
	}()

	var process []string
	var result *[]string

	// 默认登陆后的解析，不可以省略，否则后导致执行命令错位
	commandExpect(continued, stdout, ">", func(content string) {
		if content != "" {
			//fmt.Println(content) TODO 解析命令运行结果
			process = setLatestCommandResult(process, content)
		}
	})

	//_, err = stdin.Write([]byte("ls -al" + "\n"))
	//commandExpect(continued, stdout, "", func(content string) {
	//	if content != "" {
	//		//fmt.Println(content) TODO 解析命令运行结果
	//		process = setLatestCommandResult(process, content)
	//		//stopped <- true
	//	}
	//})

	_, err = stdin.Write([]byte(`echo select status from v$instance; | sqlplus / as sysdba` + "\n"))
	commandExpect(continued, stdout, ">", func(content string) {
		if content != "" {
			//fmt.Println(content)
			//prev := getPrevCommandResult(process) //prev	获取上一个命令进行联合处理判断，上一个的输出值可以是这个或者下n个的输入值
			contentBytes, _ := GbkToUtf8([]byte(content))
			content = bytes.NewBuffer(contentBytes).String()

			var data []string
			lines := strings.Split(content, "\r\n")
			lines = lines[0 : len(lines)-1]
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.Contains(line, "-----------------------------------------") {
					//fmt.Println(strconv.Quote(line))
					data = append(data, line)
				}
			}
			result = &data

			//fmt.Println(prev)
			process = setLatestCommandResult(process, content) //TODO 解析命令运行结果
			stopped <- true                                    //  最后一个命令/出现问题需要后续中断的命令 必须手动标记 停止
		}
	})

	//_, err = stdin.Write([]byte(`echo select status from v$instance; | sqlplus / as sysdba` + "\n"))
	//commandExpect(continued, stdout, ">", func(content string) {
	//	if content != "" {
	//		//fmt.Println(content)
	//		//prev := getPrevCommandResult(process) //prev	获取上一个命令进行联合处理判断，上一个的输出值可以是这个或者下n个的输入值
	//		contentBytes, _ := GbkToUtf8([]byte(content))
	//		content = bytes.NewBuffer(contentBytes).String()
	//
	//		var data []string
	//		lines := strings.Split(content, "\r\n")
	//		lines = lines[0 : len(lines)-1]
	//		for _, line := range lines {
	//			line = strings.TrimSpace(line)
	//			if line != "" && !strings.Contains(line, "-----------------------------------------") {
	//				//fmt.Println(strconv.Quote(line))
	//				data = append(data, line)
	//			}
	//		}
	//		result = &data
	//
	//		//fmt.Println(prev)
	//		process = setLatestCommandResult(process, content) //TODO 解析命令运行结果
	//		stopped <- true                                    //  最后一个命令/出现问题需要后续中断的命令 必须手动标记 停止
	//	}
	//})

	//Wait for sess to finish
	err = session.Wait()
	if err != nil {
		log.Fatal(err)
	}

	if err == nil {
		log.Println("Done! expected command to fail but it didn't")
	}
	e, ok := err.(*ssh.ExitError)
	if !ok {
		log.Printf("expected *ExitError but got %v\n", err)
	} else if e.ExitStatus() == 1 {
		log.Printf("exit code is %d\n", e.ExitStatus())
		log.Fatalf("expected command to exit with 15 but got %v", e.ExitStatus())
	}

	return *result
}

func parseOracleVerify(result string) *VerifiedDatabase {
	//logger.Appender().Infof("parse original verify result string : %s ", result)

	param := &VerifiedDatabase{}
	lines := strings.Split(result, "\r\n")
	var newLines []string

	param.Name = lines[0]
	param.Name = strings.Replace(param.Name, "\r", "", -1)
	if strings.Contains(param.Name, "\u001b[K") {
		tempName := strings.Split(param.Name, "\u001B[K")
		param.Name = tempName[2]
	}

	//param.Warn = result
	//return param

	if strings.Contains(result, "no rows selected") {
		//for _, line := range lines[1:] {
		//	if line != "" {
		//		newLines = append(newLines, line)
		//	}
		//}
		param.Warn = "no rows selected"
		return param
	}

	if strings.Contains(result, "SP2-") {
		for _, line := range lines[1:] {
			line = strings.Replace(line, "\r", "", -1)
			if line != "" && !strings.Contains(line, "SQL>") {
				newLines = append(newLines, line)
			}
		}
		fmt.Println(newLines)
		param.Warn = strings.Join(newLines, ";")
		return param
	}

	if strings.Contains(result, "ERROR at line") {
		for _, line := range lines[1:] {
			line = strings.Replace(line, "\r", "", -1)
			if line != "" {
				newLines = append(newLines, line)
			}
		}
		param.Warn = fmt.Sprintf("%s %s", newLines[2], newLines[3])
		return param
	}

	var valid = regexp.MustCompile(`^-*$`)
	for _, line := range lines[1:] {
		line = strings.Replace(line, "\r", "", -1)
		line = strings.Replace(line, "\u001b[K", "", -1)
		if line != "" && !strings.Contains(line, "--%&%--") && !strings.Contains(line, "SQL>") && !valid.MatchString(line) {
			newLines = append(newLines, line)
		}
	}

	for _, column := range strings.Split(newLines[0], "%&%") {
		column = strings.Replace(column, "\t", "", -1)
		param.Column = append(param.Column, strings.TrimSpace(column))
	}

	// TODO 删除 分页中的冗余数据（ 多出的标题连, 同时删除最后一行多扫描的 row select count
	for _, line := range newLines[1:] {
		var dataSize []string
		for _, data := range strings.Split(line, "%&%") {
			data = strings.Replace(data, "\t", "", -1)
			dataSize = append(dataSize, strings.TrimSpace(data))
		}
		param.Data = append(param.Data, dataSize)
	}

	return param
}

func main() {

	sshHost := "10.0.0.215"
	sshUser := "rdmc"
	sshPassword := "P@ssw0rd"
	sshKeyPath := ""
	sshType := "password" // password或者key
	sshPort := 56790

	// 创建ssh登录配置
	config := &ssh.ClientConfig{
		Timeout:         time.Second, // ssh连接time out时间一秒钟,如果ssh验证错误会在一秒钟返回
		User:            sshUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 这个可以,但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}

	if sshType == "password" {
		config.Auth = []ssh.AuthMethod{ssh.Password(sshPassword)}
	} else if sshType == "key" {
		config.Auth = []ssh.AuthMethod{publicKeyAuthFunc(sshKeyPath)}
	}

	// dial 获取ssh client
	addr := fmt.Sprintf("%s:%d", sshHost, sshPort)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Fatal("创建ssh client 失败", err)
	}
	defer sshClient.Close()

	// 创建ssh-session
	session, err := sshClient.NewSession()
	if err != nil {
		log.Fatal("创建ssh session失败", err)
	}

	defer session.Close()

	stopped := make(chan bool)
	continued := make(chan bool)

	var output []*VerifiedDatabase

	result := runWindowsCommand(session, stopped, continued)
	// windows oracle 比较特殊
	result = result[8 : len(result)-2]

	output = append(output, parseOracleVerify(strings.Join(result, "\r\n")))

	// ------------------------------------------------------ //
	//close(stopped)
	//close(continued)

	//for _, line := range result {
	//	//fmt.Printf("finish command group and result is %v", result)
	//	fmt.Println(line)
	//}

	fmt.Printf("%s", output)

	fmt.Println("done!")

}
