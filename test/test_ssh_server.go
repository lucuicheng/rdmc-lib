package main

import (
	"fmt"
	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

func getDate() string {
	t := time.Now()
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// 创建key 来验证 host public
func createOrLoadKeySigner() (gossh.Signer, error) {
	//key 保存到 系统temp 目录
	keyPath := filepath.Join("./", "fssh.rsa")
	//如果key 不存在则 执行 ssh-keygen 创建
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)
		//执行 ssh-keygen 创建 key
		stderr, err := exec.Command("ssh-keygen", "-f", keyPath, "-t", "rsa", "-N", "").CombinedOutput()
		output := string(stderr)
		if err != nil {
			return nil, fmt.Errorf("fail to generate private key: %v - %s", err, output)
		}
	}
	//读取文件内容
	privateBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	//生成ssh.Signer
	return gossh.ParsePrivateKey(privateBytes)
}

func setWinSize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func userAuth(user string, password string) bool {
	if user == "rdmc" && password == "P@ssw0rd" {
		return true
	} else {
		return false
	}
}

func passwordHandler(ctx ssh.Context, password string) bool {
	//check password and username
	user := ctx.User()
	return userAuth(user, password)
}

func homeHandler(s ssh.Session) {
	//tty 控制码打印彩色文字
	//mojotv.cn/tutorial/golang-term-tty-pty-vt100
	io.WriteString(s, fmt.Sprintf("\x1B[1;31mADMC 内部私有SSH通道, 当前登陆用户名: %s\x1B[0m\n", s.User()))
	io.WriteString(s, fmt.Sprintf("\x1B[0;31mVersion 2.0.1, Time : %s\x1B[0m\n", getDate()))

	cmd := exec.Command("bash")
	ptyReq, winCh, isPty := s.Pty()

	if !isPty {
		io.WriteString(s, "不是PTY请求.\n")
		s.Exit(1)
		return
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
	ptmx, err := pty.Start(cmd)
	if err != nil {
		io.WriteString(s, "bash 启动失败.\n")
		log.Println(err)
		s.Exit(1)
		return
	}
	defer func() { _ = ptmx.Close() }() // Best effort.

	//// Handle pty size.
	//ch := make(chan os.Signal, 1)
	//signal.Notify(ch, syscall.SIGWINCH)
	//go func() {
	//	for range ch {
	//		if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
	//			log.Printf("error resizing pty: %s", err)
	//		}
	//	}
	//}()
	//ch <- syscall.SIGWINCH // Initial resize.
	//defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.
	//监听终端size window 变化
	go func() {
		for win := range winCh {
			setWinSize(ptmx, win.Width, win.Height)
		}
	}()

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The exit goroutine will keep reading until the next keystroke before returning.
	go func() {
		_, _ = io.Copy(ptmx, s)
	}()
	_, _ = io.Copy(s, ptmx)

	//go func() { _, _ = io.Copy(ptmx, s) }()
	//_, _ = io.Copy(os.Stdout, s)
	//_, _ = io.Copy(os.Stdout, s)

	cmd.Wait()
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		io.WriteString(s, fmt.Sprintf("Hello %s\n", s.User()))
	})

	hostKeySigner, err := createOrLoadKeySigner()
	if err != nil {
		log.Fatal(err)
	}

	s := &ssh.Server{
		Addr:    ":56780",
		Handler: homeHandler, //
		//PublicKeyHandler:
		PasswordHandler: passwordHandler, //不需要密码验证
	}
	s.AddHostKey(hostKeySigner)
	log.Println("starting ssh server on port 56780...")
	log.Fatal(s.ListenAndServe())
}
