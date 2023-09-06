package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type MyStruct struct {
	MyTime time.Time `json:"my_time"`
}

func main() {
	jsonStr := `{"my_time":"2023-08-31T13:34:45+08:00"}`

	var myStruct MyStruct
	err := json.Unmarshal([]byte(jsonStr), &myStruct)
	if err != nil {
		fmt.Println("JSON 解析错误:", err)
		return
	}

	fmt.Println(myStruct.MyTime)
}

//package main
//
//import (
//	"fmt"
//	"time"
//)
//
//func CSTDateTime(timeStr string) time.Time {
//	var cstZone = time.FixedZone("UTC", 8*3600) // 东八区
//
//	// 将时间字符串解析为指定时区的时间类型
//	t, err := time.ParseInLocation("2006-01-02 15:04:05", timeStr, cstZone)
//	if err != nil {
//		fmt.Println("time read error：", err)
//		return time.Now()
//	}
//	return t
//}
//
//func main() {
//
//	// 待转换的时间字符串
//	timeStr := "2023-07-19 12:34:56"
//	date := CSTDateTime(timeStr)
//
//	// 输出转换后的时间
//	fmt.Println("转换后的时间：", date)
//}

//func main() {
//	filePath := "/Users/lucuicheng/Downloads/cahce/test7.tar.gz"
//
//	// 尝试打开 gzip 文件
//	file, err := os.Open(filePath)
//	if err != nil {
//		fmt.Println("Error opening file:", err)
//		return
//	}
//	defer file.Close()
//
//	// 创建 gzip 读取器
//	gzipReader, err := gzip.NewReader(file)
//	if err != nil {
//		fmt.Println("Error creating gzip reader:", err)
//		return
//	}
//	defer gzipReader.Close()
//
//
//
//	//// 尝试读取 gzip 文件头部信息
//	//
//	//if _, err := gzipReader.Peek(1); err != nil {
//	//	if err == io.EOF {
//	//		fmt.Println("Gzip file is empty.")
//	//	} else {
//	//		fmt.Println("Error reading gzip header:", err)
//	//	}
//	//	return
//	//}
//
//	fmt.Println("Gzip file can be decompressed.",gzipReader.Name)
//}
