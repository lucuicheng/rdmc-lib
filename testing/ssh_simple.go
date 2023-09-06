package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	expect "github.com/google/goexpect"
	"github.com/google/goterm/term"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"
)

const (
	timeout = 10 * time.Minute
)

func run(session *ssh.Session, command string) {
	var buff bytes.Buffer
	session.Stdout = &buff
	if err := session.Run(command); err != nil {
		log.Fatal("远程执行cmd失败", err)
	}
	fmt.Println("命令输出:", buff.String())
}

func readBuffForString(reader io.Reader, expect string) string {

	re := regexp.MustCompile(`.*#$`)

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

		//fmt.Printf("---------------|%v|%v\n", strings.TrimSpace(cleanText), re.MatchString(strings.TrimSpace(cleanText)))
		if re.MatchString(cleanText) {
			break
		}

		if err != nil {
			//fmt.Println("------------", err)
		}
	}

	return waitingString
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

func RunOneCommand(session *ssh.Session, command string) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("xterm", 400, 512, modes); err != nil {
		err = fmt.Errorf(
			"failed to request for pseudo terminal. servername: %s, err: %s",
			"dds", err)
		return
	}

	// Uncomment to store output in variable
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	if err := session.Run(command); err != nil {
		log.Fatal("远程执行cmd失败1，", err)
	}

	fmt.Println(stdoutBuf.String())
	fmt.Println(stderrBuf.String())
}

func commandExpect(continued chan bool, stdout io.Reader, expect string) {
	//fmt.Printf(readBuffForString(stdout))
	go func(stdout io.Reader) {
		content := readBuffForString(stdout, "")
		fmt.Println(content)
		continued <- true
	}(stdout)

	<-continued
}

func RunCommand(session *ssh.Session, stopped chan bool, continued chan bool) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("xterm", 400, 512, modes); err != nil {
		err = fmt.Errorf(
			"failed to request for pseudo terminal. servername: %s, err: %s",
			"dds", err)
		return
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

	_, err = stdin.Write([]byte("ls -al" + "\n"))
	commandExpect(continued, stdout, "")

	_, err = stdin.Write([]byte("whoami" + "\n"))
	commandExpect(continued, stdout, "")

	_, err = stdin.Write([]byte("exit" + "\n"))
	commandExpect(continued, stdout, "")

	//Wait for sess to finish
	err = session.Wait()
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Println(readBuffForString(stdout))

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

	// Uncomment to store in variable
	//fmt.Println(b.String())
}

func PGCommandSQL(sshClient *ssh.Client) {
	e, _, err := expect.SpawnSSH(sshClient, timeout)

	if err != nil {
		log.Printf(err.Error())
	}
	defer e.Close()

	//cmd1 := "whoami" //
	promptRE := regexp.MustCompile("\\w")

	result, _, _ := e.Expect(regexp.MustCompile("\\#"), timeout)
	e.Send("psql -U root -d momdb" + "\n")
	result1, _, _ := e.Expect(regexp.MustCompile("\\>"), timeout)
	e.Send("\\t" + "\n")
	result2, _, _ := e.Expect(promptRE, timeout)
	e.Send("\\pset pager off" + "\n")
	result3, _, _ := e.Expect(promptRE, timeout)
	e.Send("\\d" + "\n")
	result4, _, _ := e.Expect(promptRE, timeout)
	e.Send("\\q" + "\n")
	result5, _, _ := e.Expect(promptRE, timeout)
	e.Send("exit" + "\n")
	result6, _, _ := e.Expect(promptRE, timeout)

	fmt.Println(term.Greenf("Done!\n"))
	fmt.Printf("%s: result: 	%s\n", "login", result)
	fmt.Printf("%s: result 1: %s\n", "psql -U root -d momdb", result1) //correct
	fmt.Printf("%s: result 2: %s\n", "\\d", result2)                   //correct
	fmt.Printf("%s: ----------result 3: %s\n", "\\pset pager off", result3)
	fmt.Printf("%s: result 4: %s\n", "\\q", result4)
	fmt.Printf("%s: result 5:%s\n", "exit", result5)
	fmt.Printf("%s: result 6:%s\n", "exit", result6)
}

func OracleCommandSQL(sshClient *ssh.Client) {
	e, _, err := expect.SpawnSSH(sshClient, timeout)

	if err != nil {
		log.Printf(err.Error())
	}
	defer e.Close()

	//cmd1 := "whoami" //
	//promptRE := regexp.MustCompile("\\w")

	result, _, _ := e.Expect(regexp.MustCompile("\\$"), timeout)
	e.Send("su - oracle" + "\n")
	result1, _, _ := e.Expect(regexp.MustCompile("Password:"), timeout)
	e.Send("oracle" + "\n")
	result2, _, _ := e.Expect(regexp.MustCompile("\\$"), timeout)
	e.Send("sqlplus / as sysdba" + "\n")
	// 需要执行的恢复检验业务逻辑 start，只支持部分基础操作和
	result3, _, _ := e.Expect(regexp.MustCompile("\\>"), timeout)

	e.Send("select * from v$tempfile;" + "\n")
	// 需要执行的恢复检验业务逻辑 end
	result4, _, _ := e.Expect(regexp.MustCompile("\\>"), timeout)

	e.Send("select * from all_tables where owner='DDD';" + "\n")
	// 需要执行的恢复检验业务逻辑 end
	result41, _, _ := e.Expect(regexp.MustCompile("\\>"), timeout)

	e.Send("exit" + "\n") // 退出 sqlplus
	result5, _, _ := e.Expect(regexp.MustCompile("\\$"), timeout)
	e.Send("exit" + "\n") // 退出 oracle 用户
	result6, _, _ := e.Expect(regexp.MustCompile("\\$"), timeout)
	e.Send("exit" + "\n") // 退出 ssh 用户
	result7, _, _ := e.Expect(regexp.MustCompile("\\$"), timeout)

	fmt.Println(term.Greenf("Done!\n"))
	fmt.Printf("%s: result: 	%s\n", "login", result)
	fmt.Printf("%s: result 1: %s\n", "su - oracle", result1) //correct
	fmt.Printf("%s: result 2: %s\n", "oracle", result2)      //correct
	fmt.Printf("%s: result 3: %s\n", "sqlplus / as sysdba", result3)
	fmt.Printf("%s: result 4: %s\n", "select * from v$tempfile;", result4)
	fmt.Printf("%s: result 4: %s\n", "select * from all_tables where owner='SYNC';", result41)
	fmt.Printf("%s: result 5:%s\n", "exit", result5)
	fmt.Printf("%s: result 6:%s\n", "exit", result6)
	fmt.Printf("%s: result 7:%s\n", "exit", result7)
}

func BaseCommandExpect(sshClient *ssh.Client) {
	e, _, err := expect.SpawnSSH(sshClient, timeout)
	fmt.Println("start run command")

	if err != nil {
		log.Printf(err.Error())
	}
	defer e.Close()

	//cmd1 := "whoami" //
	//promptRE := regexp.MustCompile("\\w")

	result, _, _ := e.Expect(regexp.MustCompile("\\#"), timeout)
	e.Send("whoami" + "\n")
	result1, _, _ := e.Expect(regexp.MustCompile("\\#"), timeout)
	e.Send("exit" + "\n") // 退出 ssh 用户
	result7, _, _ := e.Expect(regexp.MustCompile("\\#"), timeout)

	fmt.Println(term.Greenf("Done!\n"))
	fmt.Printf("%s: --------- result: 	%s\n", "login", result)
	fmt.Printf("%s: --------- result 1: %s\n", "whoami", "1111"+strings.Replace(result1, "\r", "", -1)+"22222") //correct
	fmt.Printf("%s: --------- result 7:%s\n", "exit", "3333"+strings.Replace(result7, "\r", "", -1)+"4444")
}

func main() {
	//sshHost := "10.0.0.75"
	//sshUser := "mklop"
	//sshPassword := "mklop"
	//sshKeyPath := ""
	//sshType := "password" // password或者key
	//sshPort := 22

	//sshHost := "172.16.233.128"
	//sshUser := "rdmc"
	//sshPassword := "P@ssw0rd"
	//sshKeyPath := ""
	//sshType := "password" // password或者key
	//sshPort := 56790

	//sshHost := "10.0.0.213"
	//sshUser := "root"
	//sshPassword := "root"
	//sshKeyPath := ""
	//sshType := "password" // password或者key
	//sshPort := 22

	sshHost := "10.0.0.78"
	sshUser := "root"
	sshPort := 22
	sshPassword := "root"
	sshKeyPath := "/Users/lucuicheng/go/src/rdmc/test/id_rsa" // ssh id_rsa.id路径
	sshType := "key"                                          // password或者key

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

	//RunOneCommand(session, "ls -al")
	//RunOneCommand(session, "date")
	RunCommand(session, stopped, continued)

	// ------------------------------------------------------ //

	//OracleCommandSQL(sshClient)

	// ------------------------------------------------------ //

	fmt.Println("start command")

	//BaseCommandExpect(sshClient)

	// ------------------------------------------------------ //
}
