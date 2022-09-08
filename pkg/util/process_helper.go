package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// 检查进程pid是否在运行
func CheckPid(pid int) bool {
	pidStr := strconv.Itoa(pid)
	res, err := os.ReadFile(fmt.Sprintf("/proc/%v/comm", pidStr)) // 查看pid对应进程的命令名
	if err != nil {
		return false
	}
	pNamePath, _ := os.Executable()

	return strings.Contains(string(res), filepath.Base(pNamePath)) // 比对当前进程启动命令
}

// 当前进程已交互式方式执行外部命令
// 外部命令的标准输入输出和错误输出都重定向到当前进程
func RunCommandWithCli(cmdName string, arg ...string) int {
	cmd := exec.Command(cmdName, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
	return cmd.ProcessState.ExitCode()
}
