package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Redo日志文件头
type REDO_HEADER struct {
	Signature     [4]byte // 固定为 "ORCL"
	Version       [2]byte // 文件版本号，固定为 0x19c0
	LogBlockSize  uint32  // 日志块大小，单位为字节
	LogBlockCount uint32  // 日志块数量
	FirstChange   uint32  // 第一个日志序列号
	NextChange    uint32  // 下一个日志序列号
	FirstTime     [8]byte // 第一个日志的时间戳
	NextTime      [8]byte // 下一个日志的时间戳
}

func main() {
	f, err := os.Open("tmp/redo03.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 解析REDO文件头
	var redoHeader REDO_HEADER
	if err := binary.Read(f, binary.BigEndian, &redoHeader); err != nil {
		panic(err)
	}

	//fmt.Printf("Signature: %s\n", redoHeader.Signature)
	fmt.Printf("Version: 0x%x\n", redoHeader.Version)
	fmt.Printf("LogBlockSize: %d\n", redoHeader.LogBlockSize)
	fmt.Printf("LogBlockCount: %d\n", redoHeader.LogBlockCount)
}
