package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
)

func sendCli(name string, wg *sync.WaitGroup) {
	// 发起任意终端CLI (POST http://10.0.0.103:56790/agent/atomic/script)

	//json := []byte(`{"path": "/opt/rdmc/lib/scan -s /ddrfile /opt/rocksdb5/` + name + ` 10.0.0.142"}`)
	json := []byte(`{"path": "/bin/ls -al"}`)
	body := bytes.NewBuffer(json)

	// Create client
	client := &http.Client{}

	// Create request
	req, err := http.NewRequest("POST", "http://localhost:56790/agent/atomic/script", body)

	// Headers
	req.Header.Add("adms-job", name)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	// Fetch Request
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Failure : ", err)
		wg.Done()
		return
	}

	// Read Response Body
	respBody, _ := io.ReadAll(resp.Body)

	// Display Results
	fmt.Println(fmt.Sprintf("[%s] response Status : %s; Body : %s;", name, resp.Status, string(respBody)))
	wg.Done()
}

func main() {

	var wg sync.WaitGroup

	for i := 1; i < 6; i++ {
		wg.Add(1)
		name := strconv.Itoa(i * 1111)
		go func(name string) {
			sendCli(name, &wg)
		}(name)
	}

	wg.Wait()
	fmt.Println("all done")
}
