package netdisk

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func token(api string, token string, params []byte) ([]byte, error, time.Duration) {
	startTime := time.Now()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("url parse failed %v", err)), time.Since(startTime)
	}

	req, err := http.NewRequest(http.MethodPost, uri.String(), bytes.NewBuffer(params))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("run remote direct url, request create error %v", err)), time.Since(startTime)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Cookie", fmt.Sprintf("token=%s", token))
	}
	//req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
		return nil, errors.New(fmt.Sprintf("send failed %v", err)), time.Since(startTime)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("server returned non-OK status"), time.Since(startTime)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read response body %v", err)), time.Since(startTime)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		//log.Println("send state successfully " + string(body))
		return body, nil, time.Since(startTime)
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}

	return nil, errors.New("unknown error"), time.Since(startTime)
}

func sendFileServiceRequest(api string) ([]byte, error, time.Duration) {
	startTime := time.Now()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 120 * time.Second}

	//fmt.Printf("run database request url : %s\n", api)

	uri, err := url.Parse(api)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("url parse failed %v", err)), time.Since(startTime)
	}

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("run remote direct url, request create error %v", err)), time.Since(startTime)
	}
	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("Content-Encoding", "gzip")
	//req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error：%v", err)
		return nil, errors.New(fmt.Sprintf("send failed %v", err)), time.Since(startTime)
	}
	defer resp.Body.Close()

	//fmt.Println(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("server returned non-OK status"), time.Since(startTime)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read response body %v", err)), time.Since(startTime)
	}

	if string(body) == "" {
		//results <- fmt.Sprintf("Error fetching %s: %s", api, err.Error())
	} else {
		//log.Println("send state successfully " + string(body))
		return body, nil, time.Since(startTime)
		//results <- fmt.Sprintf("Fetched %s", fmt.Sprintf("%d;%d;%d::%s", size, 1+index*size, (index+1)*size, string(body)))
	}

	return nil, errors.New("unknown error"), time.Since(startTime)
}

func downloadPartialFile(url string, start int64, end int64, destPath string) error {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	// 设置 Range 头部，指定要下载的部分内容范围
	//req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//log.Println(resp)
		return fmt.Errorf("%v", resp)
	}

	//
	dir, _ := path.Split(destPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

type FileMD5Response struct {
	Data            string `json:"data"`
	DataDescription string `json:"dataDescription"`
	Result          string `json:"result"`
	Message         string `json:"message"`
	ClientId        string `json:"clientId"`
}

type FilePermission struct {
	EntryId             int    `json:"entryId"`
	MemberType          int    `json:"memberType"`
	MemberId            int    `json:"memberId"`
	ParentId            int    `json:"parentId"`
	ParentName          string `json:"parentName"`
	MemberName          string `json:"memberName"`
	Perm                int    `json:"perm"`
	PermFileVers        int    `json:"permFileVers"`
	PermFileAttachs     int    `json:"permFileAttachs"`
	PermCateId          int    `json:"permCateId"`
	OrigPerm            int    `json:"origPerm"`
	OrigPermFileVers    int    `json:"origPermFileVers"`
	OrigPermFileAttachs int    `json:"origPermFileAttachs"`
	OrigPermCateId      int    `json:"origPermCateId"`
	State               int    `json:"state"`
	StartTime           string `json:"startTime,omitempty"`
	OrigStartTime       string `json:"origStartTime,omitempty"`
	ExpiredTime         string `json:"expiredTime,omitempty"`
	OrigExpiredTime     string `json:"origExpiredTime,omitempty"`
}

type HiddenPermission struct {
	EntryId             int    `json:"entryId"`
	MemberType          int    `json:"memberType"`
	MemberId            int    `json:"memberId"`
	ParentId            int    `json:"parentId"`
	ParentName          string `json:"parentName"`
	MemberName          string `json:"memberName"`
	Perm                int    `json:"perm"`
	PermFileVers        int    `json:"permFileVers"`
	PermFileAttachs     int    `json:"permFileAttachs"`
	PermCateId          int    `json:"permCateId"`
	OrigPerm            int    `json:"origPerm"`
	OrigPermFileVers    int    `json:"origPermFileVers"`
	OrigPermFileAttachs int    `json:"origPermFileAttachs"`
	OrigPermCateId      int    `json:"origPermCateId"`
	State               int    `json:"state"`
	StartTime           string `json:"startTime,omitempty"`
	OrigStartTime       string `json:"origStartTime,omitempty"`
	ExpiredTime         string `json:"expiredTime,omitempty"`
	OrigExpiredTime     string `json:"origExpiredTime,omitempty"`
}

type FilePermissionData struct {
	FilePermissionList   []FilePermission
	HiddenPermissionList []HiddenPermission
}

type FilePermissionResponse struct {
	Data            FilePermissionData `json:"data"`
	DataDescription string             `json:"dataDescription"`
	Result          string             `json:"result"`
	Message         string             `json:"message"`
	ClientId        string             `json:"clientId"`
}

type DownloadCheckResponse struct {
	RegionHash string `json:"RegionHash"`
}

type Settings struct {
	PageNum    int `json:"PageNum"`
	PageSize   int `json:"PageSize"`
	TotalCount int `json:"TotalCount"`
	ViewMode   int `json:"ViewMode"`
	DocViewId  int `json:"DocViewId"`
}

type FileInfo struct {
	VirtualFullName        string           `json:"VirtualFullName"`
	FileId                 int              `json:"FileId"`
	FileName               string           `json:"FileName"`
	FileNamePath           string           `json:"FileNamePath"`
	FileCreateTime         string           `json:"FileCreateTime"`
	FileModifyTime         string           `json:"FileModifyTime"`
	FileArchiveTime        string           `json:"FileArchiveTime"`
	FileCreateOperatorName string           `json:"FileCreateOperatorName"`
	FileModifyOperatorName string           `json:"FileModifyOperatorName"`
	FileLastVerId          int              `json:"FileLastVerId"`
	FileCurVerId           int              `json:"FileCurVerId"`
	FileCurSize            int64            `json:"FileCurSize"`
	ParentFolderName       string           `json:"ParentFolderName"`
	FileMD5                string           `json:"FileMD5"`
	FilePermissions        []FilePermission `json:"FilePermissions"`
	FileAccess             string           `json:"FileAccess"`
}

type FolderInfo struct {
	VirtualFullName          string `json:"VirtualFullName"`
	FolderId                 int    `json:"FolderId"`
	FolderName               string `json:"FolderName"`
	FolderNamePath           string `json:"FolderNamePath"`
	FolderSize               int64  `json:"FolderSize"`
	FolderCreateTime         string `json:"FolderCreateTime"`
	FolderModifyTime         string `json:"FolderModifyTime"`
	FolderArchiveTime        string `json:"FolderArchiveTime"`
	FolderCreateOperatorName string `json:"FolderCreateOperatorName"`
}

type Data struct {
	FilesInfo   []FileInfo   `json:"FilesInfo"`
	FoldersInfo []FolderInfo `json:"FoldersInfo"`
	Settings    Settings     `json:"Settings"`
}

type FileAndFolderListResponse struct {
	Data Data `json:"data"`
}

type Token struct {
	Data            string `json:"data"`
	DataDescription string `json:"dataDescription"`
	Result          string `json:"result"`
	Message         string `json:"message"`
	ClientId        string `json:"clientId"`
}

// --------------------------------------------------------------------------------------------------------

type NetDisk struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func (nd NetDisk) GetToken() ([]byte, error, time.Duration) {

	api := fmt.Sprintf("http://%s:%d/api/services/Org/UserLogin", nd.Host, nd.Port)

	postParams := make(map[string]string)
	postParams["userName"] = nd.Username
	postParams["password"] = nd.Password
	paramsBytes, _ := json.Marshal(postParams)

	return token(api, "", paramsBytes)
}

func (nd NetDisk) ClearToken() ([]byte, error, time.Duration) {

	api := fmt.Sprintf("http://%s:%d/api/services/Org/UserLogout", nd.Host, nd.Port)

	postParams := make(map[string]string)
	postParams["token"] = nd.Token
	paramsBytes, _ := json.Marshal(postParams)

	return token(api, postParams["token"], paramsBytes)
}

type NetDiskFile struct {
	NetDisk  NetDisk `json:"netDisk"`
	FileId   int     `json:"fileId"`
	FileIds  string  `json:"fileIds"`
	FolderId int     `json:"folderId"`
	PageNum  int     `json:"pageNum"`
	PageSize int     `json:"pageSize"`
}

func (ndf NetDiskFile) GetInfo() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileId", strconv.Itoa(ndf.FileId))

	//for key, value := range params {
	//	query.Add(key, value)
	//}

	api := fmt.Sprintf("http://%s:%d/api/services/File/GetFileInfoById?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())
	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) GetMD5() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileId", strconv.Itoa(ndf.FileId))

	api := fmt.Sprintf("http://%s:%d/api/services/File/GetFileMd5ByFileId?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) GetPermission() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileId", strconv.Itoa(ndf.FileId))

	api := fmt.Sprintf("http://%s:%d/api/services/FilePermission/GetFilePermission?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) DownloadCheck() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileIds", strconv.Itoa(ndf.FileId))

	api := fmt.Sprintf("http://%s:%d/DownLoad/DownLoadCheck?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	// TODO 直接增加结果解析
	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) DownloadStream(regionHash string, isSingle bool, destPath string) error {

	query := url.Values{}
	query.Add("regionHash", regionHash)
	query.Add("async", "true")

	api := fmt.Sprintf("http://%s:%d/downLoad/index?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	if isSingle {
		startByte := int64(0)
		endByte := int64(1023) // 下载前 1024 字节

		//log.Println(api)

		err := downloadPartialFile(api, startByte, endByte, destPath)
		if err != nil {
			return errors.New(fmt.Sprintf("Download failed, Error: %v", err))
		} else {
			return nil
		}
	}

	return errors.New("unknown error")
	//return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) DownloadTask() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileId", strconv.Itoa(ndf.FileId))

	api := fmt.Sprintf("http://%s:%d/api/services/File/GetFileMd5ByFileId?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) DownloadCompressed() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("fileId", strconv.Itoa(ndf.FileId))

	api := fmt.Sprintf("http://%s:%d/api/services/File/GetFileMd5ByFileId?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	return sendFileServiceRequest(api)
}

func (ndf NetDiskFile) GetFileAndFolderList() ([]byte, error, time.Duration) {

	query := url.Values{}
	query.Add("token", ndf.NetDisk.Token)
	query.Add("folderId", strconv.Itoa(ndf.FolderId))
	query.Add("pageNum", strconv.Itoa(ndf.PageNum))
	query.Add("pageSize", strconv.Itoa(ndf.PageSize))

	api := fmt.Sprintf("http://%s:%d/api/services/Doc/GetFileAndFolderList?%s", ndf.NetDisk.Host, ndf.NetDisk.Port, query.Encode())

	return sendFileServiceRequest(api)
}

// --------------------------------------------------------------------------------------------------------

func SingleFileDownload(file NetDiskFile, destPath string, retryCount int) error {
	//retryCount := 0

	if retryCount > 3 {
		return errors.New("maximum number of retries reached")
	}

	if retryCount > 0 && retryCount <= 3 {
		fmt.Println("retry download ", file.FileId)
	}

	var err error
	//  单一文件 下载检测
	response, err, _ := file.DownloadCheck()
	if err != nil {
		//log.Printf("[download] failed check request %v consume [%v]\n", err, consume)
		//return err
		SingleFileDownload(file, destPath, retryCount+1)
	}

	var downloadCheckResponse DownloadCheckResponse
	json.Unmarshal(response, &downloadCheckResponse)

	//log.Printf("id is [%d] %s", file.FileId, downloadCheckResponse.RegionHash)
	//log.Printf("[download] key data is [RegionHash=%s] consume : [%v]\n", downloadCheckResponse.RegionHash, consume)
	//time.Sleep(1000 * time.Millisecond)

	//  单一文件 下载
	err = file.DownloadStream(downloadCheckResponse.RegionHash, true, destPath)
	if err != nil {
		//log.Printf("[download] failed stream request %v consume [%v]\n", err, consume)
		SingleFileDownload(file, destPath, retryCount+1)
	}

	return err
}

func resetVirtualName(netDisk NetDisk, fileAndFolderListResponse FileAndFolderListResponse, folderName string, fetchDetail bool) FileAndFolderListResponse {
	// 修改文件虚拟目录结构+获取md5，权限值
	if fetchDetail {
		var wg sync.WaitGroup // 创建一个等待组
		for i, file := range fileAndFolderListResponse.Data.FilesInfo {
			wg.Add(1)
			go func(index int, fileInfo FileInfo) {
				defer wg.Done()

				netDiskFile := NetDiskFile{NetDisk: netDisk, FileId: fileInfo.FileId}

				fileInfo.VirtualFullName += folderName + "/" + fileInfo.FileName

				var wg sync.WaitGroup // 创建一个等待组
				result := make([]interface{}, 2)

				wg.Add(1)
				go func(cache []interface{}, index int) {
					//  进行 md5 的获取
					var md5 FileMD5Response
					response, err, consume := netDiskFile.GetMD5()
					if err != nil {
						log.Fatalf("failed request %v consume [%v]\n", err, consume)
						return
					}
					json.Unmarshal(response, &md5)

					//log.Printf("md5 response : %s, consume : [%v]\n", string(response), consume)
					cache[index] = md5
					wg.Done()
				}(result, 0)

				wg.Add(1)
				go func(cache []interface{}, index int) {
					//  进行 权限 的获取
					var permission FilePermissionResponse
					response, err, consume := netDiskFile.GetPermission()
					if err != nil {
						log.Fatalf("failed request %v consume [%v]\n", err, consume)
						return
					}
					json.Unmarshal(response, &permission)

					//log.Printf("permission response : %s, consume : [%v]\n", string(response), consume)
					cache[index] = permission
					wg.Done()
				}(result, 1)

				wg.Wait()

				//log.Println(fileInfo.VirtualFullName, result[0].(FileMD5Response).Data, result[1].(FilePermissionResponse).Data)
				fileInfo.FileMD5 = result[0].(FileMD5Response).Data
				fileInfo.FilePermissions = result[1].(FilePermissionResponse).Data.FilePermissionList

				fileAndFolderListResponse.Data.FilesInfo[index] = fileInfo
			}(i, file)
		}
		wg.Wait()
	}
	// 文件结束后在统一修改 目录
	for i, folder := range fileAndFolderListResponse.Data.FoldersInfo {
		folder.VirtualFullName += folderName + "/" + folder.FolderName
		fileAndFolderListResponse.Data.FoldersInfo[i] = folder
	}

	return fileAndFolderListResponse
}

func FileAndFolderFetch(netDisk NetDisk, folderName string, folderId int) FileAndFolderListResponse {
	const (
		PageSize = 100
	)
	var fileAndFolderListResponse FileAndFolderListResponse

	// 构建顶层目录， 获取 folder 自身信息
	folder := NetDiskFile{NetDisk: netDisk, FolderId: folderId, PageNum: 1, PageSize: PageSize} // 默认按照 10 分页去读取，每次读取可控制

	response, err, consume := folder.GetFileAndFolderList()
	if err != nil {
		log.Fatalf("failed request %v consume [%v]\n", err, consume)
		return FileAndFolderListResponse{}
	}
	//log.Printf("response : %s, consume : [%v]\n", string(response), consume)
	json.Unmarshal(response, &fileAndFolderListResponse)

	// 修改虚拟目录结构
	fileAndFolderListResponse = resetVirtualName(netDisk, fileAndFolderListResponse, folderName, true)

	// 是否需要循环分页展示
	lastPage := int(math.Ceil(float64(fileAndFolderListResponse.Data.Settings.TotalCount) / float64(folder.PageSize)))

	if lastPage != 1 {
		// TODO 改并发
		for page := 2; page <= lastPage; page++ {
			folder := NetDiskFile{NetDisk: netDisk, FolderId: folderId, PageNum: page, PageSize: PageSize} // 默认按照 10 分页去读取，每次读取可控制

			response, err, _ := folder.GetFileAndFolderList()
			if err != nil {
				//log.Fatalf("failed request %v consume [%v]\n", err, consume)
				continue
			}
			var pagedFileAndFolderListResponse FileAndFolderListResponse
			json.Unmarshal(response, &pagedFileAndFolderListResponse)

			// 修改虚拟目录结构
			pagedFileAndFolderListResponse = resetVirtualName(netDisk, pagedFileAndFolderListResponse, folderName, true)

			// 增加列表数据
			fileAndFolderListResponse.Data.FilesInfo = append(fileAndFolderListResponse.Data.FilesInfo, pagedFileAndFolderListResponse.Data.FilesInfo...)
			fileAndFolderListResponse.Data.FoldersInfo = append(fileAndFolderListResponse.Data.FoldersInfo, pagedFileAndFolderListResponse.Data.FoldersInfo...)
		}
	}

	// TODO 递归查询文件夹下的所有文件，同时并发查找 并展示出完整路径

	if len(fileAndFolderListResponse.Data.FoldersInfo) > 0 { // 再读取子目录信息
		for _, folder := range fileAndFolderListResponse.Data.FoldersInfo {
			fileAndFolderList := FileAndFolderFetch(netDisk, folderName+"/"+folder.FolderName, folder.FolderId)

			fileAndFolderListResponse.Data.FilesInfo = append(fileAndFolderListResponse.Data.FilesInfo, fileAndFolderList.Data.FilesInfo...)
			fileAndFolderListResponse.Data.FoldersInfo = append(fileAndFolderListResponse.Data.FoldersInfo, fileAndFolderList.Data.FoldersInfo...)
		}
	}

	return fileAndFolderListResponse
}

func FolderFetch(netDisk NetDisk, folderName string, folderId int) FileAndFolderListResponse {
	var fileAndFolderListResponse FileAndFolderListResponse

	// 构建顶层目录， TODO 获取 folder 自身信息
	folder := NetDiskFile{NetDisk: netDisk, FolderId: folderId, PageNum: 1, PageSize: 50} // 默认按照 10 分页去读取，每次读取可控制

	response, err, consume := folder.GetFileAndFolderList()
	if err != nil {
		log.Fatalf("failed request %v consume [%v]\n", err, consume)
		return FileAndFolderListResponse{}
	}
	//log.Printf("response : %s, consume : [%v]\n", string(response), consume)
	json.Unmarshal(response, &fileAndFolderListResponse)

	// 修改虚拟目录结构
	fileAndFolderListResponse = resetVirtualName(netDisk, fileAndFolderListResponse, folderName, false)

	// 是否需要循环分页展示
	lastPage := int(math.Ceil(float64(fileAndFolderListResponse.Data.Settings.TotalCount) / float64(folder.PageSize)))

	if lastPage != 1 {
		// TODO 改并发
		for page := 2; page <= lastPage; page++ {
			folder := NetDiskFile{NetDisk: netDisk, FolderId: folderId, PageNum: page, PageSize: 10} // 默认按照 10 分页去读取，每次读取可控制

			response, err, _ := folder.GetFileAndFolderList()
			if err != nil {
				//log.Fatalf("failed request %v consume [%v]\n", err, consume)
				continue
			}
			var pagedFileAndFolderListResponse FileAndFolderListResponse
			json.Unmarshal(response, &pagedFileAndFolderListResponse)

			// 修改虚拟目录结构
			pagedFileAndFolderListResponse = resetVirtualName(netDisk, pagedFileAndFolderListResponse, folderName, false)

			// 增加列表数据
			//fileAndFolderListResponse.Data.FilesInfo = append(fileAndFolderListResponse.Data.FilesInfo, pagedFileAndFolderListResponse.Data.FilesInfo...)
			fileAndFolderListResponse.Data.FoldersInfo = append(fileAndFolderListResponse.Data.FoldersInfo, pagedFileAndFolderListResponse.Data.FoldersInfo...)
		}
	}

	// TODO 递归查询文件夹下的所有文件，同时并发查找 并展示出完整路径

	if len(fileAndFolderListResponse.Data.FoldersInfo) > 0 { // 再读取子目录信息
		for _, folder := range fileAndFolderListResponse.Data.FoldersInfo {
			fileAndFolderList := FolderFetch(netDisk, folderName+"/"+folder.FolderName, folder.FolderId)

			//fileAndFolderListResponse.Data.FilesInfo = append(fileAndFolderListResponse.Data.FilesInfo, fileAndFolderList.Data.FilesInfo...)
			fileAndFolderListResponse.Data.FoldersInfo = append(fileAndFolderListResponse.Data.FoldersInfo, fileAndFolderList.Data.FoldersInfo...)
		}
	}

	return fileAndFolderListResponse
}

type ServerCommand struct {
	Command string
	Args    []string
}

func (sc ServerCommand) Run(fetch func(line string)) {
	command := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s %s", sc.Command, strings.Join(sc.Args, " ")))
	log.Println(command.String())

	stdout, err := command.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed creating command stdoutpipe: %s", err)
		return
	}
	defer stdout.Close()
	stdoutReader := bufio.NewReader(stdout)

	stderr, err := command.StderrPipe()
	if err != nil {
		log.Fatalf("Failed creating command stderrpipe: %s", err)
		return
	}
	defer stderr.Close()
	stderrReader := bufio.NewReader(stderr)

	if err := command.Start(); err != nil {
		log.Fatalf("start raild %v", err)
		return
	}

	sc.HandleReader(fetch, stdoutReader)
	sc.HandleErrorReader(stderrReader)

	if err := command.Wait(); err != nil {
		log.Printf("start watch %d exit with err: %v\n", command.Process.Pid, err)
		if err, ok := err.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Stop Error Exit Status: %d\n", status.ExitStatus())
			}
		}
		fmt.Printf("Run Error : %s \n", err)
		return
	}
}

func (sc ServerCommand) HandleReader(filter func(line string), reader *bufio.Reader) error {
	for {
		str, err := reader.ReadString('\n')
		if len(str) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		//log := logger.Appender().WithField("pid", pid).WithField("ppid", ppid)
		filter(str)
		//if strings.Contains(str, "bait info:") {
		//	var watchedStatList []WatchedStat
		//	content := strings.Replace(str, "\n", "", -1)
		//	content = strings.Replace(content, "bait info:", "", -1)
		//	json.Unmarshal([]byte(content), &watchedStatList)
		//
		//	log.Println(fmt.Sprintf("watched state data : %s", content))
		//	//fmt.Println(watchedStatList)
		//
		//	//  转换成其他形式， 同时从本地接口中获取相信信息
		//	sendState("localhost", 56790, content) // 主机端口 需要动态传入
		//	//------------------
		//}

		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}

func (sc ServerCommand) HandleErrorReader(reader *bufio.Reader) error {
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

// --------------------------------------------------------------------------------------------------------

type TreeNode struct {
	Title    string      `json:"title"`
	Key      string      `json:"key"`
	Children []*TreeNode `json:"children,omitempty"`
}

type PathNode struct {
	Path string `json:"path"`
	Key  string `json:"key"`
}

// AddPathToTree 将路径节点添加到树结构中
func AddPathToTree(node *TreeNode, pathNode PathNode) {
	parts := strings.Split(pathNode.Path, "/")
	currentNode := node

	for _, part := range parts {
		// 检查当前节点是否包含子节点
		var foundChild *TreeNode
		for _, child := range currentNode.Children {
			if child.Title == part {
				foundChild = child
				break
			}
		}

		// 如果没有找到子节点，创建一个新的子节点
		if foundChild == nil {
			newNode := &TreeNode{
				Title: part,
				Key:   pathNode.Key,
			}
			currentNode.Children = append(currentNode.Children, newNode)
			foundChild = newNode
		}

		// 继续到下一个子节点
		currentNode = foundChild
	}
}

func CSTDateTime(timeStr string) time.Time {
	var cstZone = time.FixedZone("UTC", 8*3600) // 东八区

	// 将时间字符串解析为指定时区的时间类型
	t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, cstZone)
	if err != nil {
		fmt.Println("time read error：", err)
		return time.Now()
	}
	return t
}

func Null(file FileInfo) {

}

func RegretNetDiskWithToken(netDisk NetDisk) NetDisk {
	var result []string
	sc := ServerCommand{Command: "/opt/rdmc/lib/netdisk", Args: []string{
		"-auth",
		"-host", netDisk.Host,
		"-port", strconv.Itoa(netDisk.Port),
		"-username", netDisk.Username,
		"-password", netDisk.Password,
	}}
	sc.Run(func(line string) {
		if !strings.Contains(line, "net disk info:") {
			result = append(result, strings.Replace(line, "\n", "", -1))
		}
	})
	if len(result) == 1 {
		netDisk.Token = result[0]
	} else {
		// TODO 处理异常
	}

	return netDisk
}

func ResetNetDiskWithoutToken(netDisk NetDisk) {
	var result []string
	sc := ServerCommand{Command: "/opt/rdmc/lib/netdisk", Args: []string{
		"-auth", "-clear",
		"-token", netDisk.Token,
		"-host", netDisk.Host,
		"-port", strconv.Itoa(netDisk.Port),
		"-username", netDisk.Username,
		"-password", netDisk.Password,
	}}
	sc.Run(func(line string) {
		if !strings.Contains(line, "bait info:") {
			result = append(result, strings.Replace(line, "\n", "", -1))
		}
	})
	if len(result) == 1 {
		log.Println("token clear", result[0])
	}
}
