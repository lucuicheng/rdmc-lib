package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func parser() {
	// Extract the trailer dictionary
	//trailerString := string(trailer[start:])
	//
	//fmt.Println(start, strconv.Quote(trailerString))
	//
	//// Parse the trailer dictionary to extract metadata
	//lengthIndex := strings.Index(trailerString, "/Size ")
	//if lengthIndex == -1 {
	//	fmt.Println("Length metadata not found")
	//} else {
	//	lengthString := trailerString[lengthIndex+8:]
	//	lengthEndIndex := strings.Index(lengthString, "\n")
	//	lengthString = lengthString[:lengthEndIndex]
	//	length, err := strconv.Atoi(strings.TrimSpace(lengthString))
	//	if err != nil {
	//		fmt.Println("Error parsing length metadata")
	//	} else {
	//		fmt.Println("Length:", length)
	//	}
	//}
	//
	//nameIndex := strings.Index(trailerString, "/Root ")
	//if nameIndex == -1 {
	//	fmt.Println("Name metadata not found")
	//} else {
	//	nameString := trailerString[nameIndex+6:]
	//	nameEndIndex := strings.Index(nameString, "\n")
	//	nameString = nameString[:nameEndIndex]
	//	fmt.Println("Name:", nameString)
	//}
	//
	//widthIndex := strings.Index(trailerString, "/Info ")
	//if widthIndex == -1 {
	//	fmt.Println("Width metadata not found")
	//} else {
	//	widthString := trailerString[widthIndex+7:]
	//	widthEndIndex := strings.Index(widthString, "\n")
	//	widthString = widthString[:widthEndIndex]
	//	width, err := strconv.Atoi(strings.TrimSpace(widthString))
	//	if err != nil {
	//		fmt.Println("Error parsing width metadata")
	//	} else {
	//		fmt.Println("Width:", width)
	//	}
	//}
	//
	//heightIndex := strings.Index(trailerString, "/Height ")
	//if heightIndex == -1 {
	//	fmt.Println("Height metadata not found")
	//} else {
	//	heightString := trailerString[heightIndex+8:]
	//	heightEndIndex := strings.Index(heightString, "\n")
	//	heightString = heightString[:heightEndIndex]
	//	height, err := strconv.Atoi(strings.TrimSpace(heightString))
	//	if err != nil {
	//		fmt.Println("Error parsing height metadata")
	//	} else {
	//		fmt.Println("Height:", height)
	//	}
	//}
}

type PDF struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (p *PDF) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if p.Head != 0 {
		headSize = p.Head
	}
	var tailSize = 512
	if p.Tail != 0 {
		tailSize = p.Tail
	}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	// Print the head info
	//fmt.Printf("HEADER : %s\n", strconv.Quote(string(head)))

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "PDF") {

		// Read the file trailer
		file.Seek(int64(0-tailSize), io.SeekEnd)
		trailer := make([]byte, tailSize)
		_, err = io.ReadFull(file, trailer)
		if err != nil {
			return true, false, err
		}

		tail := string(trailer)

		if strings.Contains(tail, "Encrypt") {
			//fmt.Println("pdf is Encrypted")
			return true, true, err
		}

		return true, false, err
	}

	return false, false, err

	//// Find the start of the trailer dictionary
	//start := len(trailer) - 2
	//for trailer[start] != '<' {
	//	//fmt.Println(start, strconv.Quote(string(trailer[start])))
	//	start--
	//}
}

type XML struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (x *XML) check(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Create a byte slice to read the file's head
	const headSize = 512
	head := make([]byte, headSize)

	// Read the file's head into the byte slice
	_, err = file.Read(head)
	if err != nil && err != io.EOF {
		return false, err
	}

	// Check if the head contains an XML declaration or root element opening tag
	var xmlDeclaration struct {
		XMLName xml.Name `xml:"xml"`
	}
	err = xml.Unmarshal(head, &xmlDeclaration)
	if err == nil {
		return true, nil
	}

	var rootElement struct {
		XMLName xml.Name
	}
	err = xml.Unmarshal(head, &rootElement)
	if err == nil {
		return true, nil
	}

	return false, nil
}

type GZIP struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (g *GZIP) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if g.Head != 0 {
		headSize = g.Head
	}
	//var tailSize = 512
	//if g.Tail != 0 {
	//	tailSize = g.Tail
	//}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	/**
	\x1f\x8b\b\x00\x00\x00\x00\x00\x00\x03\xec=\xdbr\xe36\x96y\xe6W`\xed\xadr\x
	Salted__(*\xa0\xaa\x0f\x15\xf7&\x1e\xc4S\xd1\xf6\xc3\xd8\xf6r$Q@S\xa4\x9a\t\xaa\xdc\xd5-g\xce\xf9E\xf5P.=\x1e\xc6iB\xe2\xe8\x93v\x1d\xa6N\xa4\x9ai!\xdc[\x14\x8bQ
	Salted__\x92C\xdch\xa070\xe7\v\x96\x81\u0096\xe2\x10\xca\x18\x87 \xc9ѫ\x1e}\xd8\xed.%\xb27y3\xd6\xf7\x9c\xd2en5e+h\xe5\xdd{/\xe1\x05J\xb5Q\x14\xb18\xb3@
	Salted__\xbb\xc8Qs\xa75\xafPqto\xd1\xf580\x9f6\xa5\x0eJ\x93\U0007e05bƘĬ\x1b{\x9fՂFd\xe7\x12\x18\x7f\x12\xa2_\xe2\x90\xc3\"\xaa\x94O\xad}+\x15\xaf\x85
	*/
	// Print the head info
	fmt.Printf("GZIP HEADER : %s\n", strconv.Quote(string(head)))

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "\\x1f\\x8b\\b\\x00") {
		return true, false, err
	} else if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x01\\x00\\b\\x00") ||
		strings.HasPrefix(string(head), "Salted__\\") {
		return true, false, err
	}

	return false, false, err
}

type ZIP struct {
	Head int `json:"head"`
	Tail int `json:"tail"`
}

func (z *ZIP) check(path string) (bool, bool, error) {
	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 10
	if z.Head != 0 {
		headSize = z.Head
	}
	//var tailSize = 512
	//if z.Tail != 0 {
	//	tailSize = z.Tail
	//}

	// Read the first 10 bytes
	head := make([]byte, headSize)
	_, err = io.ReadFull(file, head)
	if err != nil {
		return false, false, err
	}

	// Print the head info
	//fmt.Printf("ZIP HEADER : %s\n", strconv.Quote(string(head)))

	// PK\x03\x04\x14\x00\x00\x00\b\x00\xecX  不加密
	// PK\x03\x04\x14\x00\x00\x00\b\x00
	// PK\x03\x04\x14\x00\t\x00\b\x00np\x12Uo;\xa6\xaa\xef\xb3\a\x00  加密
	// PK\x03\x04\x14\x00\t\x00\b\x00nM\xf5V\x0f\xf0Ǖ\xe4\x01\x00\x00\xe4\x02\x00\x00\x16\x00\x1c\x00co
	// PK\x03\x04\x14\x00\x00\x00\b\x00
	// PK\x03\x04\x14\x00\x01\x00\b\x00

	// Check that the file is a PDF
	if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x00\\x00\\b\\x00") {
		return true, false, err
	} else if strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\x01\\x00\\b\\x00") ||
		strings.Contains(strconv.Quote(string(head)), "PK\\x03\\x04\\x14\\x00\\t\\x00\\b\\x00") {
		return true, true, err
	}

	return false, false, err
}

type Office struct {
	Name      string `json:"path"`
	Extension string `json:"extension"`
	Head      int    `json:"head"`
	Tail      int    `json:"tail"`
}

type OfficeType string

var a = "PK\\x03\\x04\\x14\\b\\xa9T\\xf3V\\xadR\\xa5\\x91\\x95\\x01\\xca\\x06\\x13[Content_Types].xml"

const (
	OfficeWord       OfficeType = "PK\\x03\\x04\\x14\\x06\\b!ߤ\\xd2lZ\\x01 \\x05\\x13\\b\\x02[Content_Types].xml"
	OfficeExcel      OfficeType = "PK\\x03\\x04\\x14\\x06\\b!A7\\x82\\xcfn\\x01\\x04\\x05\\x13\\b\\x02[Content_Types].xml"
	OfficePowerPoint OfficeType = "PK\\x03\\x04\\x14\\x06\\b!\\xdf\\xcc\\x18\\xf5\\xad\\x01F\\f\\x13\\b\\x02[Content_Types].xml"

	OfficeDocument     OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\tdocProps/"
	OfficeSpreadsheet  OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\tdocProps/"
	OfficePresentation OfficeType = "PK\\x03\\x04\\n\\x87N\\xe2@\\x04ppt/"

	OfficePythonDoc    OfficeType = "\\xe3V\\xadR\\xa5\\x91\\x95\\x01\\xca\\x06\\x13[Content_Types].xml"
	OfficePythonSheet  OfficeType = "\\xe3VFZ\\xc1\\f\\x82\\xb1\\x10docProps/app.xml"
	OfficePythonSlider OfficeType = "\\xe3VrΪ\\xe7\\xbd\\x01<\\r\\x13[Content_Types].xml"
)

func (x *Office) check(path string, tag string, officeTypes ...OfficeType) (bool, bool, error) {

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	var headSize = 64
	if x.Head != 0 {
		headSize = x.Head
	}
	var tailSize = 960
	if x.Tail != 0 {
		tailSize = x.Tail
	}

	// 设置读取的起始位置
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		//fmt.Println("Failed to seek file:", err)
		return false, false, err
	}

	// 读取指定部分头信息
	buffer := make([]byte, headSize) // 设置读取的长度
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		//fmt.Println("Failed to read file:", err)
		return false, false, err
	}

	// 检查是否达到读取的终止位置
	if n < headSize {
		//fmt.Println("End of file reached before reading desired range")
		return false, false, err
	}

	headInfo := strconv.Quote(string(buffer[:n]))
	headInfo = strings.Replace(headInfo, "\\x00", "", -1)
	//fmt.Printf("HEADER : %s\n", headInfo)

	// 读取文件指定部分尾部信息
	file.Seek(int64(0-tailSize), io.SeekEnd)
	trailer := make([]byte, tailSize)
	_, err = io.ReadFull(file, trailer)
	if err != nil {
		return false, false, err
	}

	tail := string(trailer)
	tail = strings.Replace(strconv.Quote(string(trailer)), "\\x00", "", -1)
	//fmt.Printf("TAIL : %s\n", tail)

	for _, officeType := range officeTypes {

		if strings.Contains(headInfo, string(officeType)) || strings.Contains(tail, tag) {

			if strings.Contains(tail, "Encrypt") {
				//fmt.Println("pdf is Encrypted")
				return true, true, err
			}

			return true, false, err
		} else {

			headSize = 560

			// 设置读取的起始位置
			_, err = file.Seek(1024, io.SeekStart)
			if err != nil {
				//fmt.Println("Failed to seek file:", err)
				return false, false, err
			}

			// 读取指定部分头信息
			buffer := make([]byte, headSize) // 设置读取的长度
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				//fmt.Println("Failed to read file:", err)
				return false, false, err
			}

			// 检查是否达到读取的终止位置
			if n < headSize {
				//fmt.Println("End of file reached before reading desired range")
				return false, false, err
			}

			headInfo := strconv.Quote(string(buffer[:n]))
			headInfo = strings.Replace(headInfo, "\\x00", "", -1)
			headInfo = strings.Replace(headInfo, "\\xff", "", -1)
			//fmt.Printf("HEADER : %s\n", headInfo)

			if strings.Contains(headInfo, "EncryptedPackage") {
				//fmt.Println("pdf is Encrypted")
				return true, true, err
			}

		}
	}

	return false, false, err
}

func check(path string) (bool, string, error) {
	ext := filepath.Ext(path)

	// ---------------------------------------------------------------------------
	tag := "unmatched-suffix"
	// TODO 可优化成并发查询, 根据后缀优先调用具体 office 检测

	office := Office{Head: 512, Tail: 960}
	isOffice, isEncrypted, err := office.check(path, "word/document.xml", OfficeWord, OfficeDocument, OfficePythonDoc)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is Word/Document? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-docx"
		if ext != ".docx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".docx" {
			tag = "unmatched-suffix-docx"
		}
	}

	office = Office{Head: 512, Tail: 512}
	isOffice, isEncrypted, err = office.check(path, "worksheets/sheet1.xml", OfficeExcel, OfficeSpreadsheet, OfficePythonSheet)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is Excel/Spreadsheet? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-xlsx"
		if ext != ".xlsx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".xlsx" {
			tag = "unmatched-suffix-xlsx"
		}
	}

	office = Office{Head: 512, Tail: 512}
	isOffice, isEncrypted, err = office.check(path, "ppt/slides/slide1.xml", OfficePowerPoint, OfficePresentation, OfficePythonSlider)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isOffice {
		//fmt.Printf("is PowerPoint/Presentation? %v and pdf is encrypted? %v\n", isOffice, isEncrypted)
		tag := "matched-suffix-pptx"
		if ext != ".pptx" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".pptx" {
			tag = "unmatched-suffix-pptx"
		}
	}

	// ---------------------------------------------------------------------------

	pdf := PDF{}
	isPDF, isEncrypted, err := pdf.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isPDF {
		//fmt.Printf("is pdf? %v, and pdf is encrypted? %v is extension name corect %v\n", isPDF, isEncrypted, ext == ".pdf")
		tag := "matched-suffix-pdf"
		if ext != ".pdf" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".pdf" {
			tag = "unmatched-suffix-pdf"
		}
	}

	// ---------------------------------------------------------------------------

	zip := ZIP{Head: 32}
	isZIP, isEncrypted, err := zip.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isZIP {
		//fmt.Printf("is ZIP? %v, and zip is encrypted? %v, is extension name corect %v\n", isZIP, isEncrypted, ext == ".zip")
		tag := "matched-suffix-zip"
		if ext != ".zip" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".zip" {
			tag = "unmatched-suffix-zip"
		}
	}

	// ---------------------------------------------------------------------------

	gzip := GZIP{}
	isGZIP, isEncrypted, err := gzip.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isGZIP {
		//fmt.Printf("is GZIP? %v, and gzip is encrypted? %v\n", isGZIP, isEncrypted)
		tag := "matched-suffix-gzip"
		if ext != ".gz" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".gz" {
			tag = "unmatched-suffix-gzip"
		}
	}

	// ---------------------------------------------------------------------------

	xml := XML{}
	isXML, err := xml.check(path)
	if err != nil {
		//fmt.Printf("Error: %s\n", err)
		//return false, "", err
	}
	if isXML {
		//fmt.Printf("is XML? %v\n", isXML)
		tag := "matched-suffix-xml"
		if ext != ".xml" {
			tag = "un" + tag
		}
		return isEncrypted, tag, nil
	} else {
		if ext == ".xml" {
			tag = "unmatched-suffix-xml"
		}
	}

	return isEncrypted, tag, nil
}

func main() {

	// PK\x03\x04\x14\x00\x00\x00\b\x003Y\xe3VrΪ\xe7\xbd\x01\x00\x00<\r\x00\x00\x13\x00\x00\x00[Content_Types].xml
	//var path = "/Users/lucuicheng/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/2.0b4.0.9/d60d820cb6636bb4f51d353a88e26276/Message/MessageTemp/6db0a04772c5000f3a9dfbb886c99b25/File/aflhwoznqo.docx"
	//var path = "/Users/lucuicheng/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/2.0b4.0.9/d60d820cb6636bb4f51d353a88e26276/Message/MessageTemp/6db0a04772c5000f3a9dfbb886c99b25/File/agntjwkqwh.xlsx"
	//var path = "/Users/lucuicheng/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/2.0b4.0.9/d60d820cb6636bb4f51d353a88e26276/Message/MessageTemp/6db0a04772c5000f3a9dfbb886c99b25/File/akabqmfjth.pptx"
	//var path = "/Users/lucuicheng/Library/Containers/com.tencent.xinWeChat/Data/Library/Application Support/com.tencent.xinWeChat/2.0b4.0.9/d60d820cb6636bb4f51d353a88e26276/Message/MessageTemp/6db0a04772c5000f3a9dfbb886c99b25/File/zxqswasito.pdf"
	//var path = "/Users/lucuicheng/Downloads/cahce/report.frs"

	//var path = "/Users/lucuicheng/Downloads/cahce/enc-testzip_2.zip"
	//var path = "/Users/lucuicheng/Downloads/cahce/test2.zip"
	//var path = "/Users/lucuicheng/Downloads/cahce/test3.zip" // 不加密
	var path = "//Users/lucuicheng/Downloads/cahce/test4.zip" // 不加密
	//var path = "/Users/lucuicheng/Downloads/cahce/hoppscotch-3.0.1.tar.gz" // 加密

	flag, tag, _ := check(path)
	fmt.Println(flag, tag)
}
