package main

import (
	"bufio"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"rdmc/pkg"
	"strings"
	"sync"
	"sync/atomic"
)

type FMD5 struct {
	FilePath string
	Md5Sum   string
	Count    int64
}

var checkedCount int64
var notFoundPathCount int64

func calculateMD5(filePath string, results chan<- FMD5, wg *sync.WaitGroup) bool {
	defer wg.Done()

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file %s: %v\n", filePath, err)
		atomic.AddInt64(&notFoundPathCount, 1)
		return false
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		log.Printf("Failed to calculate MD5 for file %s: %v\n", filePath, err)
		return false
	}

	md5Sum := fmt.Sprintf("%x", hash.Sum(nil))
	atomic.AddInt64(&checkedCount, 1)
	results <- FMD5{filePath, md5Sum, checkedCount}
	return true
}

func countLines(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return int64(0), err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := int64(0)
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return int64(0), err
	}

	return lineCount, nil
}

func main() {
	//Define the root directory to scan
	sourcePath := flag.String("source", "tmp/123456789-deduplication-1684145230872.txt", "The md5 check source file path")
	targetPath := flag.String("target", "/opt/rdmc/tmp/md5_default.txt", "The md5 filteredCount file output path")
	threshold := flag.Int("threshold", -1, "The threshold at which duplicate file occurrences need to be flagged")
	version := flag.Bool("v", false, "Prints current tool version")

	flag.Parse()
	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - File MD5 Filter v%s", pkg.AppVersion))
		os.Exit(0)
	}

	// 读取文件路径列表
	//sourcePath := "tmp/123456789-deduplication-1684145230872.txt"
	file, err := os.Open(*sourcePath)
	if err != nil {
		log.Fatal("Failed to open file_paths.txt:", err)
	}
	defer file.Close()

	// 同步读取文件，预先获取文件行数，同时检测文件可读性
	// 启动多个 goroutine 来计算文件的 MD5 值
	lineCount, err := countLines(*sourcePath)
	if err != nil {
		log.Fatal("Error:", err)
		return
	}
	fmt.Println("lineCount: ", lineCount)

	var wg sync.WaitGroup // 创建一个等待组

	maxConcurrency := 5                           // 最大并发打开文件数
	results := make(chan FMD5)                    // 创建一个通道来接收 MD5 结果
	fileChan := make(chan string, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

	// 启动多个 goroutine 处理文件
	for i := 0; i < maxConcurrency-1; i++ {
		go func() {
			for filePath := range fileChan {
				wg.Add(1)
				calculateMD5(filePath, results, &wg)
			}
		}()
	}

	go func() {
		// 启动多个 goroutine 来计算文件的 MD5 值
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			filePath := scanner.Text()
			fileChan <- filePath
		}
		close(fileChan) // 关闭通道，告知处理文件的 goroutine 没有更多任务

	}()

	// 等待所有计算完成
	//wg.Add(1)
	//go func() {
	//	results <- FMD5{FilePath: "sssssss", Count: int64(-1)} // 结束标志位
	//	wg.Done()
	//}()
	wg.Wait()

	//done := make(chan string)
	md5Map := make(map[string][]string)
	done := make(chan string)
	go func() {
		for {
			select {
			case fmd5, _ := <-results:
				//atomic.AddInt64(&resultCount, 1)
				//fmt.Println(fmd5, notFoundPathCount+fmd5.Count, lineCount)
				_, fileName := path.Split(fmd5.FilePath) // 解析路径
				key := fmd5.Md5Sum + ":" + fileName      // 组合成 key
				md5Map[key] = append(md5Map[key], fmd5.FilePath)
				if lineCount == notFoundPathCount+fmd5.Count {
					done <- "done"
					return
				}
			default:

			}
		}
	}()

	fmt.Println(<-done)

	targetPathDir, _ := path.Split(*targetPath)

	cmd := exec.Command("mkdir", "-p", targetPathDir)
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
		return
	}

	// 完整结束指定进程
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Child command %d exit with err: %v\n", cmd.Process.Pid, err)
		return
	}

	// 最后写入 目标文件
	// 创建文件，如果文件已存在则会覆盖
	file, err = os.Create(*targetPath)
	if err != nil {
		log.Printf("file create failed : %v\n", err)
		defer file.Close()
		return
	}
	defer file.Close()

	// 创建一个写入缓冲区
	writer := bufio.NewWriter(file)

	// 输出重复的 MD5 值及其数量  写入文件的每一行内容
	for key, paths := range md5Map {
		keys := strings.Split(key, ":")
		//md5Sum := keys[0]
		fileName := keys[1]

		if len(paths) > 1 {
			if len(paths) >= *threshold { // 超过阈值才进行计算
				//_, err := writer.WriteString(fmt.Sprintf("RANSOMWARE: %s(%s); Count: %d\r\n", fileName, md5Sum, len(paths)))
				_, err := writer.WriteString(fmt.Sprintf("RANSOMWARE: %s, Count: %d\r\n", fileName, len(paths)))
				// 循环写入具体路径
				for _, path := range paths {
					writer.WriteString(path + "\r\n")
				}
				writer.WriteString("\r\n")
				if err != nil {
					log.Printf("data write failed : %v\n", err)
				}
				//_, err = writer.WriteString(fmt.Sprintf("Paths: %v\r\n", paths))
				//if err != nil {
				//	log.Printf("data write failed : %v\n", err)
				//}
				//_, err = writer.WriteString(fmt.Sprintf("Count: %d\r\n", len(paths)))
				//if err != nil {
				//	log.Printf("data write failed : %v\n", err)
				//}
			}
		}
	}

	// 刷新缓冲区，确保所有数据都写入文件
	err = writer.Flush()
	if err != nil {
		log.Printf("file flush failed : %v\n", err)
	}
}
