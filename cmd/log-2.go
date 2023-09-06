package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Redo日志文件头
type redo_header struct {
	Signature [5]byte // RDO\x00
	Version   uint32  // 版本号,19c为0x06000005
	BlockSize uint16  // 一般为512字节
	NumBlocks uint32  // 文件块数
}

// 变更记录头
type change_header struct {
	Type    uint16 // 1 - 数据,2 - 回退,3 - 事务
	DataLen uint16 // 数据长度
	RecLen  uint16 // 记录长度,包含下面3个字段
	SQLLen  uint16 // SQL长度
	SCN     uint64 // 变更SCN
	SqlData []byte // SQL语句
}

func main() {
	f, err := os.Open("tmp/redo01.log")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	//解析REDO文件头
	var redoHeader redo_header
	if err := binary.Read(f, binary.BigEndian, &redoHeader); err != nil {
		panic(err)
	}

	//定位到第一个日志记录
	if _, err := f.Seek(int64(redoHeader.BlockSize), 0); err != nil {
		panic(err)
	}

	for {
		//解析变更记录头
		var changeHeader change_header
		if err := binary.Read(f, binary.BigEndian, &changeHeader); err != nil {
			fmt.Println("sdssss")
			break
		}

		//读取SQL语句
		sqlData := make([]byte, changeHeader.SQLLen)
		if _, err := f.Read(sqlData); err != nil {
			panic(err)
		}
		sql := string(sqlData)
		fmt.Printf("SQL: %s\n", sql)

		//跳过数据区
		if _, err := f.Seek(int64(changeHeader.RecLen-26), 1); err != nil {
			panic(err)
		}
	}
}
