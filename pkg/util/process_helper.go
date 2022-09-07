package util

import (
	"fmt"
	"os"
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
