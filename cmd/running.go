package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func main() {
	pid := 12345 // 进程ID

	// 获取进程的创建时间戳
	creationTime, err := getProcessCreationTime(pid)
	if err != nil {
		fmt.Println("获取进程创建时间失败:", err)
		os.Exit(1)
	}

	// 获取当前时间
	currentTime := time.Now()

	// 计算进程的存活时间
	uptime := currentTime.Sub(creationTime)

	fmt.Println("进程存活时间:", uptime)
}

func getProcessCreationTime(pid int) (time.Time, error) {
	// 打开进程的stat文件
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	file, err := os.Open(statPath)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	// 读取stat文件的内容
	var (
		state                                                          string
		ppid                                                           int
		pgrp, session                                                  int
		ttyNr                                                          int
		tpgid                                                          int
		flags                                                          uint
		minflt, cminflt, majflt, cmajflt, utime, stime, cutime, cstime int
		startTime                                                      uint64
	)
	_, err = fmt.Fscanf(file, "%d %s %d %d %d %d %d %d %d %d %d %d %d %d %d",
		&pid, &state, &ppid, &pgrp, &session, &ttyNr, &tpgid, &flags,
		&minflt, &cminflt, &majflt, &cmajflt, &utime, &stime, &cutime, &cstime, &startTime)
	if err != nil {
		return time.Time{}, err
	}

	// 计算进程的启动时间
	bootTime, err := getSystemBootTime()
	if err != nil {
		return time.Time{}, err
	}
	startTimeSec := float64(startTime) / float64(syscall.Sysconf(syscall.SC_CLK_TCK))
	startTimeNano := int64(startTimeSec*1000000000) + int64(bootTime.UnixNano())

	return time.Unix(0, startTimeNano), nil
}

func getSystemBootTime() (time.Time, error) {
	// 打开/proc/uptime文件
	uptimePath := "/proc/uptime"
	file, err := os.Open(uptimePath)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	// 读取系统启动时间
	var uptimeSec float64
	_, err = fmt.Fscanf(file, "%f", &uptimeSec)
	if err != nil {
		return time.Time{}, err
	}

	// 计算系统启动时间
	bootTime := time.Now().Add(-time.Duration(uptimeSec) * time.Second)

	return bootTime, nil
}
