package constant

import (
	"fmt"
	"os/user"
	"runtime"
)

func env(path string) string {
	sysType := runtime.GOOS

	// 获取当前用户信息
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("can not read user:", err)
		return "/opt" + path
	}

	// 用户名
	username := currentUser.Username

	if err != nil {
		fmt.Printf("%v\n", err)
	}

	if sysType == "darwin" {
		return "/opt" + path
	} else if sysType == "linux" {
		if username == "root" {
			return "/opt" + path
		}

		return ".cmba" + path // 非 root 用户的 隐含日志文件位置
	} else {
		return "." // 应该是 Windows 的
	}
}

// TODO 增加当前运行用户 判断，如果是 非 root 用户自定切换配置文件路径

var BinPath = env("/rdmc/bin")       //注意这里不能带有 /
var ConfigPath = env("/rdmc/etc")    //注意这里不能带有 /
var LogPath = env("/rdmc/logs/")     //注意这里需要带有 /
var TmpPath = env("/rdmc/tmp")       //注意这里 不 需要带有
var UploadPath = env("/rdmc/upload") //注意这里 不 需要带有
