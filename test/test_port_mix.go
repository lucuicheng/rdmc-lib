package main

//
//import (
//	"rdmc/pkg"
//	pb "rdmc/test/service/helloworld"
//	"context"
//	"fmt"
//	"github.com/creack/pty"
//	"github.com/gin-gonic/gin"
//	"github.com/gliderlabs/ssh"
//	"github.com/soheilhy/cmux"
//	gossh "golang.org/x/crypto/ssh"
//	"golang.org/x/term"
//	"google.golang.org/grpc"
//	"io"
//	"io/ioutil"
//	"log"
//	"net"
//	"net/http"
//	"os"
//	"os/exec"
//	"path/filepath"
//	"syscall"
//	"time"
//	"unsafe"
//)
//
////--------------------------------------- ssh
//
////创建key 来验证 host public
//func createOrLoadKeySignerNew() (gossh.Signer, error) {
//	//key 保存到 系统temp 目录
//	keyPath := filepath.Join("./", "fssh.rsa")
//	//如果key 不存在则 执行 ssh-keygen 创建
//	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
//		os.MkdirAll(filepath.Dir(keyPath), os.ModePerm)
//		//执行 ssh-keygen 创建 key
//		stderr, err := exec.Command("ssh-keygen", "-f", keyPath, "-t", "rsa", "-N", "").CombinedOutput()
//		output := string(stderr)
//		if err != nil {
//			return nil, fmt.Errorf("fail to generate private key: %v - %s", err, output)
//		}
//	}
//	//读取文件内容
//	privateBytes, err := ioutil.ReadFile(keyPath)
//	if err != nil {
//		return nil, err
//	}
//	//生成ssh.Signer
//	return gossh.ParsePrivateKey(privateBytes)
//}
//
//func setWinSizeNew(f *os.File, w, h int) {
//	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
//		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
//}
//
//func userAuthNew(user string, password string) bool {
//	if user == "rdmc" && password == "P@ssw0rd" {
//		return true
//	} else {
//		return false
//	}
//}
//
//func passwordHandlerNew(ctx ssh.Context, password string) bool {
//	//check password and username
//	user := ctx.User()
//	return userAuthNew(user, password)
//}
//
//func homeHandlerNew(s ssh.Session) {
//	//tty 控制码打印彩色文字
//	//mojotv.cn/tutorial/golang-term-tty-pty-vt100
//	io.WriteString(s, fmt.Sprintf("\x1B[1;31mADMC 内部私有SSH通道, 当前登陆用户名: %s\x1B[0m\n", s.User()))
//	io.WriteString(s, fmt.Sprintf("\x1B[0;31mVersion 2.0.1, Time : %s\x1B[0m\n", pkg.GetDate()))
//
//	cmd := exec.Command("bash")
//	ptyReq, winCh, isPty := s.Pty()
//
//	if !isPty {
//		io.WriteString(s, "不是PTY请求.\n")
//		s.Exit(1)
//		return
//	}
//
//	cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
//	ptmx, err := pty.Start(cmd)
//	if err != nil {
//		io.WriteString(s, "bash 启动失败.\n")
//		log.Println(err)
//		s.Exit(1)
//		return
//	}
//	defer func() { _ = ptmx.Close() }() // Best effort.
//
//	//// Handle pty size.
//	//ch := make(chan os.Signal, 1)
//	//signal.Notify(ch, syscall.SIGWINCH)
//	//go func() {
//	//	for range ch {
//	//		if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
//	//			log.Printf("error resizing pty: %s", err)
//	//		}
//	//	}
//	//}()
//	//ch <- syscall.SIGWINCH // Initial resize.
//	//defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.
//	//监听终端size window 变化
//	go func() {
//		for win := range winCh {
//			setWinSizeNew(ptmx, win.Width, win.Height)
//		}
//	}()
//
//	// Set stdin in raw mode.
//	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
//	if err != nil {
//		panic(err)
//	}
//	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.
//
//	// Copy stdin to the pty and the pty to stdout.
//	// NOTE: The exit goroutine will keep reading until the next keystroke before returning.
//	go func() {
//		_, _ = io.Copy(ptmx, s)
//	}()
//	_, _ = io.Copy(s, ptmx)
//
//	//go func() { _, _ = io.Copy(ptmx, s) }()
//	//_, _ = io.Copy(os.Stdout, s)
//	//_, _ = io.Copy(os.Stdout, s)
//
//	cmd.Wait()
//}
//
////--------------------------------------- grpc
//
//// server is used to implement helloworld.GreeterServer.
//type server struct {
//	pb.UnimplementedGreeterServer
//}
//
//// SayHello implements helloworld.GreeterServer
//func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
//	log.Printf("Received: %v", in.GetName())
//	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
//}
//
//func main() {
//	// Create the main listener.
//	l, err := net.Listen("tcp", ":23456")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create a cmux.
//	m := cmux.New(l)
//
//	// Match connections in order:
//	// First grpc, then HTTP, and otherwise Go RPC/TCP.
//	/**
//	Java gRPC Clients: Java gRPC client blocks until it receives a SETTINGS frame from the server.
//	If you are using the Java client to connect to a cmux'ed gRPC server please match with writers:
//	*/
//	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
//	httpL := m.Match(cmux.HTTP1Fast())
//	sshL := m.Match(cmux.Any()) // Any means anything that is not yet matched.
//
//	//------------------------ http server
//
//	engine := gin.New()
//
//	engine.GET("/user/:name", func(c *gin.Context) {
//		name := c.Param("name")
//		c.String(http.StatusOK, "Hello %s", name)
//	})
//
//	// However, this one will match /user/john/ and also /user/john/send
//	// If no other routers match /user/john, it will redirect to /user/john/
//	engine.GET("/user/:name/*action", func(c *gin.Context) {
//		name := c.Param("name")
//		action := c.Param("action")
//		message := name + " is " + action
//		c.String(http.StatusOK, message)
//	})
//
//	httpS := &http.Server{
//		Handler:        engine,
//		ReadTimeout:    120 * time.Second,
//		WriteTimeout:   120 * time.Second,
//		MaxHeaderBytes: 1 << 20,
//	}
//	//httpS.ListenAndServe()
//
//	//------------------------ ssh server
//
//	ssh.Handle(func(s ssh.Session) {
//		io.WriteString(s, fmt.Sprintf("Hello %s\n", s.User()))
//	})
//
//	hostKeySigner, err := createOrLoadKeySignerNew()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	sshS := &ssh.Server{
//		Handler: homeHandlerNew, //
//		//PublicKeyHandler:
//		PasswordHandler: passwordHandlerNew, // 默认密码验证
//	}
//	sshS.AddHostKey(hostKeySigner)
//
//	//------------------------ grpc server
//
//	grpcS := grpc.NewServer()
//	pb.RegisterGreeterServer(grpcS, &server{})
//
//	// Use the muxed listeners for your servers.
//	go httpS.Serve(httpL)
//	go sshS.Serve(sshL)
//	go grpcS.Serve(grpcL)
//
//	// Start serving!
//	m.Serve()
//}
