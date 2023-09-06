package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

func isXMLFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
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

func main() {
	filePath := "/Users/lucuicheng/Downloads/cahce/emp_docx_file/1.type.DIR_SIGNIFIER.xml"

	isXML, err := isXMLFile(filePath)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	if isXML {
		fmt.Println("The file is XML.")
	} else {
		fmt.Println("The file is not XML.")
	}
}
