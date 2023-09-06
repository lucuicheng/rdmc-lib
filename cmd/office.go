package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

func main() {

	fmt.Println("sdd")

	// Open the file
	file, err := os.Open("/Users/lucuicheng/Downloads/cahce/加密-word.docx")
	//file, err := os.Open("/Users/lucuicheng/Downloads/cahce/ccc.pptx")
	//file, err := os.Open("/Users/lucuicheng/Downloads/cahce/eocejusvys_emp.docx")
	//file, err := os.Open("/Users/lucuicheng/Downloads/cahce/eocejusvys_pdf.docx")
	//file, err := os.Open("/Users/lucuicheng/Downloads/cahce/eocejusvys.docx")
	if err != nil {
		fmt.Println("read file failed")
		return
	}
	defer file.Close()

	// Read the first 10 bytes
	head := make([]byte, 1024)
	_, err = io.ReadFull(file, head)
	if err != nil {
		fmt.Println("read file failed")
		return
	}

	// Print the head info
	fmt.Printf("HADER : %s\n", strconv.Quote(string(head)))

	// Check that the file is a PDF
	//if !strings.Contains(strconv.Quote(string(head)), "PDF") {
	//	fmt.Println("File is not a PDF")
	//	return
	//}

	//// Read the file trailer
	//file.Seek(-1024, io.SeekEnd)
	//trailer := make([]byte, 1024)
	//_, err = io.ReadFull(file, trailer)
	//if err != nil {
	//	panic(err)
	//}

	// Read the file trailer
	file.Seek(-1024, io.SeekEnd)
	trailer := make([]byte, 1024)
	_, err = io.ReadFull(file, trailer)
	if err != nil {
		panic(err)
	}

	fmt.Println("--------------")
	fmt.Printf("%s\n", string(trailer))
	fmt.Println("--------------")

	// Find the start of the trailer dictionary
	start := len(trailer) - 1
	for trailer[start] != ':' {
		//fmt.Println(start, strconv.Quote(string(trailer[start])))
		start--
		if start < 0 {
			start = 0
			break
		}
	}

	fmt.Println("running...")

}
