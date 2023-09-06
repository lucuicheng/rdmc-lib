package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("scan.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	suffixes := make(map[string]int)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ".")
		if len(parts) > 1 {
			suffix := strings.ToLower(parts[len(parts)-1])
			suffixes[suffix]++
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		return
	}

	for suffix, count := range suffixes {
		fmt.Printf("%s: %d\n", suffix, count)
	}
}
