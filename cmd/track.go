package main

func main() {
	////find /proc -mindepth 3 -maxdepth 3 -type l | awk -F/ '$4 == "fd"'
	//cmd := exec.Command("find", "/proc", "-mindepth", "3", "-maxdepth", "3", "-type", "")
	//if err := cmd.Start(); err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//
	//// 完整结束指定进程
	//if err := cmd.Wait(); err != nil {
	//	fmt.Printf("Child command %d exit with err: %v\n", cmd.Process.Pid, err)
	//	return
	//}
}
