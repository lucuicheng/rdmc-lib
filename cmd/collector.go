package main

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"rdmc/pkg"
	"strings"
	"sync"
	"time"
)

type Copied struct {
	Logs   []string `json:"logs"`
	Failed []string `json:"failed"`
}

type LogCollectedResponse struct {
	Body Copied `json:"body"`
	Code int
	Msg  string
}

func zipFiles(filename string, files []string) error {
	zipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		err := addToZip(zipWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addToZip(zipWriter *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// 修改压缩文件的名称，去掉文件夹前缀
	header.Name = filepath.Base(filename)

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// downloadRequest 请求下载文件
func downloadRequest(host string, action string, formValues url.Values) (map[string]interface{}, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	api := fmt.Sprintf("http://%s:56790/agent/atomic/%s", host, action)

	//fmt.Printf("run remote download url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	//formValues := url.Values{}
	//formValues.Set("request", "localhost:56790")
	//formValues.Set("filePath", "/tmp/tmp/ssss.sas")
	//formValues.Set("output", "/tmp/ssss.sas")
	formDataStr := formValues.Encode()

	//fmt.Printf("run remote download url, and the params is %s\n", formDataStr)

	req, err := http.NewRequest(http.MethodPost, uri.String(), strings.NewReader(formDataStr))
	if err != nil {
		fmt.Printf("run remote download url, request create error %v\n", err)
		//panic(err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		//panic(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		return nil, err
	}

	if string(body) == "" {
		//fmt.Println(errors.New(fmt.Sprintf(" atomic internal Request Failed, Send %s", api)))
		return nil, errors.New(fmt.Sprintf(" atomic internal Request Failed, Send %s", api))
	}

	resetBody := toJsonObject(string(body))

	return resetBody, nil
}

// getSHA256HashCode hash 值计算方法
func getSHA256HashCode(message []byte, cut int) string {
	hash := sha256.New()
	hash.Write(message)
	bytes := hash.Sum(nil)
	hashCode := hex.EncodeToString(bytes)
	return strings.Join(strings.Split(hashCode, "")[0:cut], "")
}

// toJsonObject 转换为对象
func toJsonObject(result string) map[string]interface{} {
	var data map[string]interface{} // 对象
	//var dats []map[string]interface{} // 数组

	err := json.Unmarshal([]byte(result), &data)
	if err != nil {
		fmt.Println(err.Error())
	}
	return data
}

func sendLogCollect(hosts string, dates string, names string) (interface{}, error) {
	// 发起任意终端CLI (POST http://10.0.0.103:56790/agent/atomic/script)

	// Create client
	client := &http.Client{}

	// Create request
	api := fmt.Sprintf("http://localhost:56790/agent/atomic/log_collection?hosts=%s&dates=%s&names=%s", hosts, dates, names)
	uri, err := url.Parse(api)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error send by url : %s", err))
	}
	log.Println(uri.String())
	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)

	// Headers
	//req.Header.Add("adms-job", name)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	// Fetch Request
	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading response body from : %s", err))
	}

	// Read Response Body
	respBody, _ := io.ReadAll(resp.Body)

	// Display Results
	var copied LogCollectedResponse
	json.Unmarshal(respBody, &copied)
	return copied, nil
}

func main() {
	version := flag.Bool("v", false, "Prints current tool version")

	hosts := flag.String("h", "", "grep log file contained name")
	dates := flag.String("d", "", "grep log file contained name")
	names := flag.String("n", "", "grep log file contained name")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - Log Collection Operator v%s", pkg.AppVersion))
		os.Exit(0)
	}

	start := time.Now()

	if *names != "" {
		//var segments []string
		var wg sync.WaitGroup

		suffix := getSHA256HashCode([]byte(strings.Join([]string{time.Now().String()}, "")), 12) // 随机生成 临时 后缀

		// 获取需要打包的远程文件地址，调用 http 地址，搜索->复制->列表
		logs, err := sendLogCollect(*hosts, *dates, *names)
		if err != nil {
			fmt.Println("run failed ,", err)
			return
		}

		response := logs.(LogCollectedResponse)
		results := make(chan string, len(response.Body.Logs))

		var files []string

		// 按照 列表 下载文件，生成压缩包
		log.Printf("collected log file count [%d], and list is :\n", len(response.Body.Logs))
		for _, logPath := range response.Body.Logs {
			log.Printf("		%s\n", logPath)
		}

		for _, segment := range response.Body.Logs {
			wg.Add(1)

			go func(segment string) {
				defer wg.Done()

				seq := strings.SplitN(segment, "-", 2) // 只按照第一个 - 划分
				address := seq[0] + ":56790"
				filePath := strings.Replace(seq[1], "/opt/rdmc", "", -1)

				formValues := url.Values{}

				formValues.Set("request", address) // 应该是远程 客户端IP 地址
				formValues.Set("filePath", filePath)
				_, fileName := path.Split(filePath)
				target := fmt.Sprintf("/opt/rdmc/tmp/collected/%s/%s_%s", suffix, seq[0], fileName)
				formValues.Set("output", target) // TODO 需要补完完整路径

				response, err := downloadRequest("localhost", "download", formValues)

				log.Printf("download [%s] from [%s] response %s\n", filePath, address, response)

				if err != nil {
					//pkg.RunFailed(err, c)
					//fmt.Printf("Error Send download tmp file failed : %s\n", err)
					results <- fmt.Sprintf("Error Send download tmp file failed : %s\n", err)
					return
				}

				if response["body"] != nil {
					m := response["body"].(map[string]interface{})
					if m["status"].(float64) != 0 {
						//pkg.RunFailed(err, c)
						//fmt.Printf("execute download tmp file failed : %s\n", err)
						results <- fmt.Sprintf("Error execute download tmp file failed : %s", err)
						return
					}
					results <- m["dist"].(string)
				} else {
					//pkg.RunFailed(err, c)
					//fmt.Printf("request download tmp file failed : %s\n", err)
					results <- fmt.Sprintf("Error request download tmp file failed : %s", err)
					return
				}
			}(segment)
		}

		wg.Wait()
		close(results)

		for result := range results {
			filePath := result
			if !strings.HasPrefix(filePath, "Error") {
				files = append(files, filePath)
			}
		}

		timestamp := time.Now().Format("20060102150405")
		zipFilename := fmt.Sprintf("/opt/rdmc/tmp/clogs_%s@%s.zip", *names, timestamp)

		err = zipFiles(zipFilename, files)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			//fmt.Println("Files compressed successfully.")
			fmt.Println(zipFilename)
		}

	}

	log.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
}
