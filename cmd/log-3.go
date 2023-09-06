package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

type RedoLogHeader struct {
	LogFileSize        uint64
	CheckSum           uint32
	LogFileSequence    uint32
	LowScn             uint64
	HighScn            uint64
	NextLowScn         uint64
	NextHighScn        uint64
	OnlineLog          uint8
	CheckSumAlgorithm  uint8
	CompressionEnabled uint8
	EncryptionEnabled  uint8
	Pad                uint8
	Fill               [3]byte
}

type RedoRecordHeader struct {
	Len        uint16
	Type       uint16
	Lwn        uint32
	Hwn        uint32
	Seq        uint32
	Crc        uint16
	Fill1      uint16
	LowScn     uint64
	HighScn    uint64
	Pdb        uint64
	Spare      uint64
	FirstTime  uint64
	LastTime   uint64
	ResetStamp uint64
	Fill2      [8]byte
}

type RedoRecord struct {
	Header RedoRecordHeader
	Data   []byte
}

type RedoRecordType uint16

const (
	// Redo record types
	RedoLogData       RedoRecordType = 0x05
	RedoBlockData     RedoRecordType = 0x06
	RedoBlockChecksum RedoRecordType = 0x07
	RedoEndOfFile     RedoRecordType = 0x0b
	RedoGeneric       RedoRecordType = 0x0c
)

func main() {

	// Open the redo log file
	file, err := os.Open("tmp/redo01.log")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the redo log header
	var header RedoLogHeader
	err = binary.Read(file, binary.LittleEndian, &header)
	if err != nil {
		panic(err)
	}

	// Print the redo log header
	fmt.Println("Redo Log Header:")
	fmt.Printf("  LogFileSize: %d\n", header.LogFileSize)
	fmt.Printf("  CheckSum: %d\n", header.CheckSum)
	fmt.Printf("  LogFileSequence: %d\n", header.LogFileSequence)
	fmt.Printf("  LowScn: %d\n", header.LowScn)
	fmt.Printf("  HighScn: %d\n", header.HighScn)
	fmt.Printf("  NextLowScn: %d\n", header.NextLowScn)
	fmt.Printf("  NextHighScn: %d\n", header.NextHighScn)
	fmt.Printf("  OnlineLog: %d\n", header.OnlineLog)
	fmt.Printf("  CheckSumAlgorithm: %d\n", header.CheckSumAlgorithm)
	fmt.Printf("  CompressionEnabled: %d\n", header.CompressionEnabled)
	fmt.Printf("  EncryptionEnabled: %d\n", header.EncryptionEnabled)
	fmt.Printf("  Pad: %d\n", header.Pad)

	// 打开redo日志文件
	file, err = os.Open("tmp/redo01.log")
	if err != nil {
		fmt.Println("Failed to open redo log file:", err)
		return
	}
	defer file.Close()

	// 每个日志条目有固定的字节大小
	const entrySize = 512

	count := 0
	// 读取文件中的每个日志条目
	for {
		// 读取512字节的数据块
		data := make([]byte, entrySize)
		_, err := file.Read(data)
		if err != nil {
			// 如果已经到达文件末尾，则退出循环
			if err.Error() == "EOF" {
				break
			}

			// 否则出现了错误
			fmt.Println("Failed to read redo log entry:", err)
			return
		}

		// 解析文件头信息
		//version := data[0:2]
		//fileSize := data[2:8]

		if count == 0 {
			// 读取 8 个字节长度的标识符
			signature := string(data[0:8])
			version := binary.BigEndian.Uint16(data[8:10])
			fileSize := binary.BigEndian.Uint32(data[10:14])
			fileStatus := binary.BigEndian.Uint32(data[14:18])
			//blockSize := binary.BigEndian.Uint32(data[18:22])
			//
			//// 输出解析结果
			fmt.Printf("Head signature: %d, Version: %d fileSize: %d, File Status:%d\n", signature, version, fileSize, fileStatus)
		} else if count == 1 {
			//// 解析SCN号和时间戳
			scn := binary.BigEndian.Uint64(data[4:12])
			timestamp := binary.BigEndian.Uint32(data[12:16])
			timeObj := time.Unix(int64(timestamp), 0)
			//
			//// 输出解析结果
			fmt.Printf("Block Header SCN: %d, Timestamp: %s\n", scn, timeObj)
		} else {
			//// 解析SCN号和时间戳
			scn := binary.BigEndian.Uint64(data[4:12])
			timestamp := binary.BigEndian.Uint32(data[12:16])
			timeObj := time.Unix(int64(timestamp), 0)
			//
			//// 输出解析结果
			fmt.Printf("No:%d, Type: %s, SCN: %d, Timestamp: %s\n", count-1, "sa", scn, timeObj)
		}

		count++
		if count == 5 {
			break
		}

	}
}

//package main
//
//import (
//	"encoding/binary"
//	"fmt"
//	"os"
//)
//
//func main() {
//	// 打开redo日志文件
//	file, err := os.Open("tmp/redo01.log")
//	if err != nil {
//		fmt.Println("Failed to open redo log file:", err)
//		return
//	}
//	defer file.Close()
//
//	//scanner := bufio.NewScanner(file)
//	//count := 0
//	//for scanner.Scan() {
//	//	count++
//	//	line := scanner.Text()
//	//	fmt.Println(count, ":", line)
//	//	if count == 10 {
//	//		break
//	//	}
//	//}
//
//	// 读取固定长度的头部
//	header := make([]byte, 16)
//	_, err = file.Read(header)
//	if err != nil {
//		fmt.Println("Failed to read header:", err)
//		return
//	}
//
//	// 解析SCN号
//	scn := binary.BigEndian.Uint64(header[4:12])
//
//	// 输出SCN号
//	fmt.Println("SCN:", scn)
//}
