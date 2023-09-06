package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"net/url"
	"rdmc/internal/machine"
	"rdmc/pkg"
	"rdmc/pkg/constant"
	"rdmc/pkg/logger"
	"strings"
	"time"
)

func readConfig(log *logrus.Entry) (pkg.Config, error) {
	config := pkg.Config{}

	configFile := fmt.Sprintf("%s/%s", constant.ConfigPath, "server.yml") // 默认配置文件路径

	data, fileErr := ioutil.ReadFile(configFile)
	if fileErr != nil {
		log.Errorf("config error: %v", fileErr)
		return config, fileErr
	}

	log.Infof("Reading %s", configFile)
	err := yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		log.Errorf("error: %v", err)
		return config, err
	}

	return config, nil
}

func heartbeatRequest(host string, port int, ctx context.Context, log *logrus.Entry) (*http.Transport, *http.Client, *http.Request) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}
	api := fmt.Sprintf("http://%s:%d/agent/index/heartbeat", host, port)

	log.Infof("run remote download url : %s", api)

	uri, err := url.Parse(api)
	if err != nil {
		fmt.Println(err.Error())
		return nil, nil, nil
	}

	config, _ := readConfig(log)
	realIP, _ := machine.GetLocalIP() // 获取本地的 IP 地址
	config.Server.Address = realIP

	paramsBytes, _ := json.Marshal(config.Server)

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(string(paramsBytes)))
	if err != nil {
		log.Errorf("run remote direct url, request create error %v", err)
		return nil, nil, nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.WithContext(ctx)

	return tr, client, req
}

func heartbeatSender(client *http.Client, req *http.Request) (map[string]interface{}, error) {
	var empty map[string]interface{}

	resp, err := client.Do(req)

	if err != nil {
		return empty, err
	}

	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		return empty, err
	}

	if string(bytes) == "" {
		//fmt.Println(errors.New(fmt.Sprintf(" atomic internal Request Failed, Send %s", api)))
		return empty, errors.New(fmt.Sprintf(" heartbeat internal Request Failed, Send"))
	}

	resetBody := pkg.ToJSONObject(string(bytes))

	return resetBody, nil
}

func testCancel() {
	timestamp := time.Now().Format("15:04:05")

	address := "10.0.0.152"
	port := 56790

	log := logger.Appender().
		WithField("component", "MANAGER").
		WithField("category", "CLIENT-NODE")

	start1 := time.Now()
	//，重试 三次 即为 下线 offline, 同时处理超时请求，超过 5, 主动 取消请求，确保不会堆积
	phase1 := make(chan bool)
	done := make(chan bool)

	cx, cancel := context.WithCancel(context.Background())
	_, client, req := heartbeatRequest(address, port, cx, log)

	go func(phase1 chan bool, done chan bool,
		timestamp string, log *logrus.Entry, address string, port int) {

		//start1 := time.Now()
		phaseStatus := true

		response, err := heartbeatSender(client, req)
		if err != nil {
			//fmt.Printf("[%s:%d] %s heartbeat request consume : [%v] "+
			//	"and err is %v\n", address, port, timestamp, time.Since(start1), err)
			log.Errorf("[%s:%d] master server connect failed, the port is closed,"+
				" please confirm the firewall is opened or not also check the serve is started or not",
				address, port)
			phaseStatus = false // 请求异常
		} else {
			heartbeatResponse := response["body"]
			// //ctx, cancelFunc := context.WithCancel(context.Background())
			//fmt.Println("heartbeat", heartbeatResponse)
			if heartbeatResponse == nil {
				log.Errorf("[%s:%d] master server get heartbeat response failed, "+
					"please check the version is same or not",
					address, port)
				phaseStatus = false
			} else {
				// TODO 记录最新 的心跳记录 详细信息
			}

			//fmt.Printf("[%s:%d] %s heartbeat request consume : [%v]\n", address, port, timestamp, time.Since(start1))

		}

		select {
		case phase1 <- phaseStatus:
		default:
			return
		}

		// 阶段2 需要增加的操作

		done <- true
	}(phase1, done, timestamp, log, address, port)

	select {
	case status := <-phase1:
		<-done
		if status {
			//fmt.Println("done")
			fmt.Println(pterm.Green(fmt.Sprintf("[%s:%d] %s heartbeat request consume : [%v]",
				address, port, timestamp, time.Since(start1))))
		} else { // 心跳异常情况
			fmt.Println(pterm.Red(fmt.Sprintf("[%s:%d] %s heartbeat request consume : [%v] "+
				"some internal error",
				address, port, timestamp, time.Since(start1))))
		}

	case <-time.After(time.Second * 2): // 心跳请求发送超时 默认为 2s,非 http request 的 超时时间，request 超时时间为 120s
		fmt.Println(pterm.Red(fmt.Sprintf("[%s:%d] %s heartbeat request consume : [%v] "+
			"heartbeat timeout",
			address, port, timestamp, time.Since(start1))))
		cancel()
		fmt.Println("取消数据")
		//retry(retries, timestamp, log, address, port) // 触发超时重试
	}
}

func main() {
	for i := 0; i < 3; i++ {
		go testCancel()
	}
	select {}
}
