package main

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const ACTIVEIO = "Activeio-"

func decrypt(encrypt string) string {
	var result strings.Builder
	for _, char := range encrypt {
		decryptedChar := rune(char + 6)
		result.WriteRune(decryptedChar)
	}

	fmt.Println(result.String())

	decodedStr, _ := base64.StdEncoding.DecodeString(result.String())
	decryptedStr := strings.ReplaceAll(string(decodedStr), ACTIVEIO, "")

	return decryptedStr
}

func main() {
	encryptedStr := "root"
	decryptedStr := decrypt(encryptedStr)
	fmt.Println(decryptedStr)
}
