package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// Define the root directory to scan
	diskName := flag.String("disk", "C", "the disk name, suche as 'C'")

	flag.Parse()

	root := fmt.Sprintf("%s:\\", *diskName)

	// Define the list of file extensions to filter
	extFilter := []string{}

	// Create a channel to receive file paths
	files := make(chan string)

	// Create a channel to signal when all workers are done
	done := make(chan bool)

	// Define the number of worker goroutines to use
	numWorkers := 10

	start := time.Now()

	// Start the workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for filePath := range files {
				ext := filepath.Ext(filePath) // TODO 增加非后缀类型的
				// TODO 内部再次增加 异步循环，比对多个 extFilter 组
				for _, filter := range extFilter {
					if ext == filter[1:] {
						fmt.Printf("TARGET is %s\n", filePath)
						break
					}
				}
			}
			done <- true
		}()
	}

	// Traverse the root directory recursively and send file paths to the channel
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("ERROR is %v\n", err)
			//return err
		}
		if !info.IsDir() {
			//fmt.Println("----", path)
			files <- path
		}
		return nil
	})
	if err != nil {
		fmt.Printf("ERROR is %v\n", err)
	}

	// Close the channel to signal no more files will be sent
	close(files)

	// Wait for all workers to finish
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	fmt.Printf("Finish Tasks Cost=[%v]\n", time.Since(start))
}
