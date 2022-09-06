package util

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// 检查进程pid是否在运行
func CheckPid(pid int) bool {
	if runtime.GOOS == "windows" {
		_, err := os.FindProcess(pid)
		return err == nil
	}

	pidStr := strconv.Itoa(pid)
	res, err := os.ReadFile(fmt.Sprintf("/proc/%v/comm", pidStr)) // 查看pid对应进程的命令名
	if err != nil {
		return false
	}
	pNamePath, _ := os.Executable()

	return strings.Contains(string(res), filepath.Base(pNamePath)) // 比对当前进程启动命令

	// cmd := exec.Command("kill", "-0", pidStr)
	// err := cmd.Start()
	// if err != nil {
	// 	return false
	// }
	// err = cmd.Wait()
	// if err != nil {
	// 	return false
	// }
	// if cmd.ProcessState != nil {
	// 	if cmd.ProcessState.ExitCode() == 0 {
	// 		return true
	// 	}
	// }
	// return false
}
