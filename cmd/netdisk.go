package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"rdmc/internal/netdisk"
	"rdmc/pkg"
	"strconv"
	"sync"
	"sync/atomic"
)

func singleFileDetection(netDisk netdisk.NetDisk, fileId int) {
	// 构建单一文件
	file := netdisk.NetDiskFile{NetDisk: netDisk, FileId: fileId}

	//  单一文件先获取 基础信息
	response, err, consume := file.GetInfo()
	if err != nil {
		log.Fatalf("failed request %v consume [%v]\n", err, consume)
		return
	}
	var fileInfo netdisk.FileInfo
	json.Unmarshal(response, &fileInfo)
	log.Printf("response : %s, consume : [%v]\n", fileInfo.FileNamePath, consume)

	//  单一文件再获取 md5
	response, err, consume = file.GetMD5()
	if err != nil {
		log.Fatalf("failed request %v consume [%v]\n", err, consume)
		return
	}
	log.Printf("response : %s, consume : [%v]\n", string(response), consume)

	//  单一文件再获取 permission
	response, err, consume = file.GetPermission()
	if err != nil {
		log.Fatalf("failed request %v consume [%v]\n", err, consume)
		return
	}
	log.Printf("response : %s, consume : [%v]\n", string(response), consume)

	// TODO 文件元数据保存

	// TODO 文件下载
	err = netdisk.SingleFileDownload(file, "aaaaaaaa.pptx", 0)
	if err != nil {
		return
	}

	// TODO 实时检测文件
	// TODO 返回检测结果
}

func main() {

	auth := flag.Bool("auth", false, "")
	clear := flag.Bool("clear", false, "")
	synchronize := flag.Bool("sync", false, "")
	structs := flag.Bool("structs", false, "")
	check := flag.Bool("check", false, "")

	host := flag.String("host", "localhost", "the net disk running host")
	port := flag.Int("port", 30179, "the net disk running port")
	username := flag.String("username", "admin", "the net disk running host")
	password := flag.String("password", "edoc2edoc2", "the net disk task running host")

	token := flag.String("token", "", "")

	path := flag.String("path", "PublicRoot", "")
	id := flag.Int("id", 1, "")

	version := flag.Bool("v", false, "Prints current tool version")

	flag.Parse()

	if *version {
		fmt.Println(fmt.Sprintf("ActiveIO CLI - FRS NetDisk Fetch v%s", pkg.AppVersion))
		os.Exit(0)
		return
	}

	// 初始化对象
	netDisk := netdisk.NetDisk{Host: *host, Port: *port, Username: *username, Password: *password}

	// 生成 token
	if *auth && !*clear {
		response, err, consume := netDisk.GetToken()
		if err != nil {
			log.Fatalf("failed request %v consume [%v]\n", err, consume)
		}
		log.Printf("response : %s, consume : [%v]\n", string(response), consume)

		var token netdisk.Token
		json.Unmarshal(response, &token)

		fmt.Println(token.Data)

		return
	}

	// 清除 token
	if *auth && *clear {
		netDisk.Token = *token
		// 校验
		response, err, consume := netDisk.ClearToken()
		if err != nil {
			log.Fatalf("failed request %v consume [%v]\n", err, consume)
		}
		log.Printf("response : %s, consume : [%v]\n", string(response), consume)

		var token netdisk.Token
		json.Unmarshal(response, &token)

		fmt.Println(token.Data)
	}

	// 网盘目录结构
	if *structs {
		log.Printf("net disk info is %v\n", netDisk)

		// 从缓存中获取 token  抽取公共方法
		if *token == "" {
			netDisk = netdisk.RegretNetDiskWithToken(netDisk)
		} else {
			netDisk.Token = *token
		}
		log.Printf("token is %s\n", netDisk.Token)

		// 文件目录下的所有内容获取
		//folderList := fileAndFolderFetch(netDisk, "PublicRoot", 1)
		folderListResponse := netdisk.FolderFetch(netDisk, *path, *id) // 默认同步的路径

		// 构建树结构
		root := &netdisk.TreeNode{
			Title: "Root",
			Key:   "0",
		}

		netdisk.AddPathToTree(root, netdisk.PathNode{Path: *path, Key: strconv.Itoa(*id)})
		for _, folder := range folderListResponse.Data.FoldersInfo {
			//bytesString, _ := json.Marshal(folder)
			//fmt.Println(string(bytesString))
			netdisk.AddPathToTree(root, netdisk.PathNode{Path: folder.VirtualFullName, Key: strconv.Itoa(folder.FolderId)})
			//fmt.Println(folder.VirtualFullName, folder.FolderId)
		}

		// 转换为 JSON
		jsonData, err := json.Marshal(root)
		if err != nil {
			fmt.Println("JSON 编码失败:", err)
			return
		}

		fmt.Println(string(jsonData))

		//  及时登出 token 抽取公共方法
		netdisk.ResetNetDiskWithoutToken(netDisk)

		return
	}

	// 同步文件信息
	if *synchronize {

		var count int64
		// TODO 从缓存中获取 token
		if *token == "" {
			netDisk = netdisk.RegretNetDiskWithToken(netDisk)

		} else {
			netDisk.Token = *token
		}
		log.Printf("token is %s\n", netDisk.Token)

		// 文件目录下的所有内容获取
		fileAndFolderList := netdisk.FileAndFolderFetch(netDisk, *path, *id) // 默认同步的路径

		// TODO 复制原始 list + 从上次检测结果中过滤为缓存和检测的

		// 增减原子应用的 atomic 计数 并发下载
		var wg sync.WaitGroup // 创建一个等待组
		maxConcurrency := 1   // 最大并发打开文件数
		//results := make(chan error)                     // 创建一个通道来接收 重复数据 结果
		fileChan := make(chan netdisk.FileInfo, maxConcurrency) // 使用有缓冲的通道限制并发打开的文件数

		for i := 0; i < maxConcurrency; i++ {
			wg.Add(1)
			go func() {
				for file := range fileChan {
					//err := singleFileDownload(NetDiskFile{NetDisk: netDisk, FileId: file.FileId}, file.VirtualFullName)
					//if err != nil {
					//	log.Printf("file: %s downloaded failed, id is [%d], %v\n", file.VirtualFullName, file.FileId, err)
					//	// TODO 队列优化机制， 同样使用递归，直到结束位置
					//} else {
					//	fmt.Printf("downloaded file: %s, id is [%d]\n", file.VirtualFullName, file.FileId)
					//}
					//fmt.Printf("downloaded file: %s, id is [%d]\n", file.VirtualFullName, file.FileId)
					netdisk.Null(file)
					//results <- err
				}
				wg.Done()
			}()
		}

		go func() {
			for _, file := range fileAndFolderList.Data.FilesInfo {
				fmt.Printf("file: %s, id is [%d], md5 is [%s] and permission is %v\n", file.VirtualFullName, file.FileId, file.FileMD5, file.FilePermissions)
				atomic.AddInt64(&count, 1)
				fileChan <- file
			}
			//for _, folder := range fileAndFolderList.Data.FoldersInfo {
			//	fmt.Printf("folder: %s, id is [%d]\n", folder.VirtualFullName, folder.FolderId)
			//}
			close(fileChan)
		}()

		wg.Wait()

		fmt.Println("all done", atomic.LoadInt64(&count))

		//  及时登出 token
		netdisk.ResetNetDiskWithoutToken(netDisk)

		return
	}

	if *check {
		// TODO 从缓存中获取 token
		netDisk = netdisk.RegretNetDiskWithToken(netDisk)

		if netDisk.Token != "" {
			log.Printf("token is %s\n", netDisk.Token)
			fmt.Println("checked: successful")

			//  及时登出 token
			netdisk.ResetNetDiskWithoutToken(netDisk)

		} else {
			// TODO 处理异常
			fmt.Println("checked: failed")
		}

	}
	// x

	// 单文件检测
	if false {
		//
		//singleFileDetection(netDisk, 88)
	}
}
