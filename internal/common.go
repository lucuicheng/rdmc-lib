package internal

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"rdmc/internal/constant"
	"strings"
	"time"
)

func Asset(name string) ([]byte, error) {
	// 模拟读取文件
	file, err := os.Open(name)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("open failed")
	}
	defer file.Close()

	// Get the file size
	stat, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("state failed")
	}

	// Read the file into a byte slice
	bs := make([]byte, stat.Size())
	_, err = bufio.NewReader(file).Read(bs)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return nil, errors.New("read failed")
	}

	return bs, nil
}

type WatchedStat struct {
	Path       string `json:"path"`
	Ip         string `json:"ip"`
	Pid        int    `json:"pid"`
	ServerPort int    `json:"serverPort"`
	ClientPort int    `json:"clientPort"`
	FilePath   string `json:"filePath"`
	AttackTime string `json:"attackTime"`
	Operation  string `json:"operation"`
	CmdLine    string `json:"cmdLine"`
}

type Items struct {
	Items []map[string]int `json:"items"`
}

type ItemsResponse struct {
	Body Items
	Code int
	Msg  string
}

// CreateAllDir 调用os.MkdirAll递归创建文件夹
func createAllDir(filePath string) error {
	if !IsExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			fmt.Println("创建文件夹失败,error info:", err)
			return err
		}
		return err
	}
	return nil
}

// IsExist 判断所给路径文件/文件夹是否存在(返回true是存在)
func IsExist(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func changeFileMtime(filePath string, newTime time.Time) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	accessTime := fileInfo.ModTime()
	err = os.Chtimes(filePath, accessTime, newTime)
	if err != nil {
		return err
	}

	return nil
}

// wg *sync.WaitGroup
func CreatBait(sourceFile string, targetPathDir string) {

	defer func() {
		//wg.Done()
		if err := recover(); err != nil {
			fmt.Printf("error %v\n", err)
			return
		}
	}()

	data, err := Asset(fmt.Sprintf("/opt/rdmc/lib/assets/bait/%s", sourceFile))
	if err != nil {
		log.Println(err)
		// Asset was not found.
		return
	}

	// 构建目标文件完整路径
	targetPath := path.Join(targetPathDir, sourceFile)

	createAllDir(targetPathDir)

	// 最后写入 目标文件
	// 创建文件，如果文件已存在则会覆盖
	file, err := os.Create(targetPath)
	defer file.Close()

	if err != nil {
		log.Printf("file create failed : %v\n", err)
		return
	}

	// 创建一个写入缓冲区
	writer := bufio.NewWriter(file)
	reader := strings.NewReader(string(data))
	//  优化多次写入
	for {
		buffer := make([]byte, 1024)
		readCount, readErr := reader.Read(buffer)
		if readErr == io.EOF {
			break
		} else {
			writer.Write(buffer[:readCount])
		}
	}

	// 刷新缓冲区，确保所有数据都写入文件
	err = writer.Flush()
	if err != nil {
		log.Printf("file flush failed : %v\n", err)
	}

	// 修改文件权限
	err = os.Chmod(targetPath, 0666)
	if err != nil {
		log.Printf("chmod to [%s] failed\n", targetPath)
	}

	if strings.Contains(targetPath, "___aaa-password-rec") {
		// 修改文件 mtime 2012-01-01 00:01:15
		newTime := time.Date(2012, time.January, 1, 0, 1, 15, 0, time.FixedZone("CST", 8*60*60))

		err := changeFileMtime(targetPath, newTime)
		if err != nil {
			fmt.Println("Failed to change mtime:", err)
			return
		}
	}

	fmt.Println(targetPath)
	//wg.Done()
}

func HandleReader(reader *bufio.Reader) error {
	for {
		str, err := reader.ReadString('\n')
		if len(str) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		//log := logger.Appender().WithField("pid", pid).WithField("ppid", ppid)

		if strings.Contains(str, "bait info:") {
			var watchedStatList []WatchedStat
			content := strings.Replace(str, "\n", "", -1)
			content = strings.Replace(content, "bait info:", "", -1)
			json.Unmarshal([]byte(content), &watchedStatList)

			log.Println(fmt.Sprintf("watched state data : %s", content))
			//fmt.Println(watchedStatList)

			//  转换成其他形式， 同时从本地接口中获取相信信息
			sendState("localhost", 56790, content) // 主机端口 需要动态传入
			//------------------
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func handleProcessReader(reader *bufio.Reader) error {
	for {
		str, err := reader.ReadString('\n')
		if len(str) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		//log := logger.Appender().WithField("pid", pid).WithField("ppid", ppid)

		content := strings.Replace(str, "\n", "", -1)
		fmt.Println(content)

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func HandleErrorReader(reader *bufio.Reader) error {
	for {
		str, err := reader.ReadString('\n')
		if len(str) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		//log := logger.Appender().WithField("pid", pid).WithField("ppid", ppid)
		//log.Println(strings.Replace(str, "\n", "", -1))

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func RunProcessDetect(watchedPath string, shouldPrint bool) []string {
	// args to command
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var cmd *exec.Cmd

	//不指定用户时，直接运行
	cmd = exec.CommandContext(ctx, "/bin/sh", "-c",
		fmt.Sprintf("ps -ef | grep -v grep | grep 'baits -watch ' | grep '%s' | awk '{print $2}' ", watchedPath))

	//log.Println(cmd.String())

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("normally inner logger error : %v\n", err)
			return
		}
	}()

	if ctx.Err() == context.DeadlineExceeded {
		if shouldPrint {
			fmt.Println("Command Timed Out")
		}
		//return errors.New("command timed out"), "", ""
		os.Exit(1)
		return []string{"Command Timed Out"}
	}

	if err != nil {
		if stderr.String() != "" {
			if shouldPrint {
				fmt.Printf(stderr.String())
			}
			os.Exit(1)
			return strings.Split(stderr.String(), "\n")
		}
		if shouldPrint {
			fmt.Println(err)
		}
		os.Exit(1)
		return strings.Split(err.Error(), "\n")
	}
	if shouldPrint {
		fmt.Print(stdout.String())
	}
	return strings.Split(stdout.String(), "\n")
}

func RunSubProcessDetect(watchedPath string, shouldPrint bool) []string {
	// args to command
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var cmd *exec.Cmd

	//不指定用户时，直接运行
	cmd = exec.CommandContext(ctx, "/bin/sh", "-c",
		fmt.Sprintf("ps -ef | grep -v grep | grep 'bait -m ' | grep '%s' | awk '{print $2}' ", watchedPath))

	//log.Println(cmd.String())

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("normally inner logger error : %v\n", err)
			return
		}
	}()

	if ctx.Err() == context.DeadlineExceeded {
		if shouldPrint {
			fmt.Println("Command Timed Out")
		}
		//return errors.New("command timed out"), "", ""
		os.Exit(1)
		return []string{"Command Timed Out"}
	}

	if err != nil {
		if stderr.String() != "" {
			if shouldPrint {
				fmt.Printf(stderr.String())
			}
			os.Exit(1)
			return strings.Split(stderr.String(), "\n")
		}
		if shouldPrint {
			fmt.Println(err)
		}
		os.Exit(1)
		return strings.Split(err.Error(), "\n")
	}
	if shouldPrint {
		fmt.Print(stdout.String())
	}
	return strings.Split(stdout.String(), "\n")
}

func RunProcessWatching(shouldPrint bool) []string {
	// args to command
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var cmd *exec.Cmd

	//不指定用户时，直接运行
	cmd = exec.CommandContext(ctx, "/bin/sh", "-c",
		fmt.Sprintf("ps -ef | grep -v grep | grep 'baits -watched-list' | awk '{print $2}' "))

	//log.Println(cmd.String())

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("normally inner logger error : %v\n", err)
			return
		}
	}()

	if ctx.Err() == context.DeadlineExceeded {
		if shouldPrint {
			fmt.Println("Command Timed Out")
		}
		//return errors.New("command timed out"), "", ""
		os.Exit(1)
		return []string{"Command Timed Out"}
	}

	if err != nil {
		if stderr.String() != "" {
			if shouldPrint {
				fmt.Printf(stderr.String())
			}
			os.Exit(1)
			return strings.Split(stderr.String(), "\n")
		}
		if shouldPrint {
			fmt.Println(err)
		}
		os.Exit(1)
		return strings.Split(err.Error(), "\n")
	}
	if shouldPrint {
		fmt.Print(stdout.String())
	}
	return strings.Split(stdout.String(), "\n")
}

func BaitExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err

}

func sendState(host string, port int, params string) {

	api := fmt.Sprintf("http://%s:%d/agent/atomic/bait_state", host, port)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {

	}

	req, err := http.NewRequest(http.MethodPut, uri.String(), strings.NewReader(params))
	if err != nil {
		fmt.Printf("run remote direct url, request create error %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		fmt.Println("send state successfully " + string(body))
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}
}

func SendStatus(host string, port int, params string) {
	api := fmt.Sprintf("http://%s:%d/agent/atomic/bait_status", host, port)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {

	}

	req, err := http.NewRequest(http.MethodPut, uri.String(), strings.NewReader(params))
	if err != nil {
		fmt.Printf("run remote direct url, request create error %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		//log.Println("send status successfully " + string(body))
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}
}

func Register(host string, port int, token string, params string) {
	api := fmt.Sprintf("http://%s:%d/agent/atomic/startup", host, port)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {

	}

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(params))
	if err != nil {
		fmt.Printf("run remote direct url, request create error %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Authorization", token)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		log.Println("register successfully " + string(body))
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}
}

func Erase(host string, port int, params string) {
	api := fmt.Sprintf("http://%s:%d/agent/atomic/startup", host, port)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {

	}

	req, err := http.NewRequest(http.MethodDelete, uri.String(), strings.NewReader(params))
	if err != nil {
		fmt.Printf("run remote direct url, request create error %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		log.Println("erase successfully " + string(body))
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}
}

func List(host string, port int, params string) ItemsResponse {
	var startUpItems ItemsResponse
	api := fmt.Sprintf("http://%s:%d/agent/atomic/startup", host, port)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		return startUpItems
	}

	req, err := http.NewRequest(http.MethodGet, uri.String(), strings.NewReader(params))
	if err != nil {
		fmt.Printf("run remote direct url, request create error %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error：%v", err)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
		return startUpItems
	} else {
		//log.Println("list successfully " + string(body))
		json.Unmarshal(body, &startUpItems)
		return startUpItems
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}
}

type Server struct {
	ID      string
	Address string
	Port    int
}

type Middleware struct {
	Name    string
	Enabled bool
}

type Route struct {
	Name    string
	Enabled bool
	Setting map[string]interface{}
	Service []Service
}

type Service struct {
	Name    string
	Enabled bool
	Setting map[string]interface{}
}

type Config struct {
	Name        string
	Version     string
	Author      string
	Company     string
	Description string
	Email       string
	Log         string
	Server      Server
	Masters     []Server
	Middleware  []Middleware
	Route       []Route
}

func ReadConfig() (Config, error) {
	//log *logrus.Entry
	config := Config{}

	configFile := fmt.Sprintf("%s/%s", constant.ConfigPath, "server.yml") // 默认配置文件路径

	data, fileErr := os.ReadFile(configFile)
	if fileErr != nil {
		//log.Errorf("config error: %v", fileErr)
		fmt.Printf("config error: %v\n", fileErr)
		return config, fileErr
	}

	//log.Infof("Reading %s", configFile)
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		//log.Errorf("error: %v", err)
		fmt.Printf("error: %v\n", err)
		return config, err
	}

	return config, nil
}
