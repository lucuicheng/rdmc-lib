package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

func main() {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var arguments []string

	arguments = append(arguments, "-host")      // 运行的具体命令
	arguments = append(arguments, "10.0.0.103") // 运行的具体命令
	arguments = append(arguments, "-port")      // 运行的具体命令
	arguments = append(arguments, "1521")       // 运行的具体命令
	arguments = append(arguments, "-username")  // 运行的具体命令
	arguments = append(arguments, "dong1")      // 运行的具体命令
	arguments = append(arguments, "-password")  // 运行的具体命令
	arguments = append(arguments, "root")       // 运行的具体命令
	arguments = append(arguments, "-sid")       // 运行的具体命令
	arguments = append(arguments, "leora")      // 运行的具体命令
	arguments = append(arguments, "-structs")   // 运行的具体命令

	cmd := exec.CommandContext(ctx, "oracle", arguments...)
	//fmt.Println(cmd.String())

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("normally inner logger error : %v", err)
		}
	}()

	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Print(stdout.String(), stderr.String())
}
