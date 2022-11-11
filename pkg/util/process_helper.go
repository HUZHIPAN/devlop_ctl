package util

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var ShowDebugInfo = false

// 检查进程pid是否在运行
func CheckProcessIsRunning(pid int, processName string) bool {
	pidStr := strconv.Itoa(pid)
	res, err := os.ReadFile(fmt.Sprintf("/proc/%v/comm", pidStr)) // 查看pid对应进程的命令名
	if err != nil {
		return false
	}

	return strings.Contains(string(res), processName) // 比对当前进程启动命令
}

// 获取进程命令行启动参数
func GetProcessCmdlineParams(pid int) string {
	pidStr := strconv.Itoa(pid)
	res, err := os.ReadFile(fmt.Sprintf("/proc/%v/cmdline", pidStr)) // 查看pid对应进程的命令名
	if err != nil {
		return ""
	}

	cmdlineStr := strings.ReplaceAll(string(res), "\000", " ");
	return cmdlineStr
}


// 当前进程已交互式方式执行外部命令
// 外部命令的标准输入输出和错误输出都重定向到当前进程
func RunCommandWithCli(cmdName string, arg ...string) int {
	if ShowDebugInfo {
		fmt.Printf("run command cli：%v %v \n", cmdName, arg)
	}
	cmd := exec.Command(cmdName, arg...)
	cmd.Env = []string{""}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return cmd.ProcessState.ExitCode()
}

// 执行命令，后台执行
func RunCommandWithDaemon(cmdName string, arg ...string) (int,error) {
	if ShowDebugInfo {
		fmt.Printf("run command daemon：%v %v \n", cmdName, arg)
	}

	cmd := exec.Command(cmdName, arg...)
	cmd.Stdin = nil
	cmd.Stderr = nil
	cmd.Stdout = nil
	err := cmd.Start()
	return cmd.Process.Pid, err
}

// 执行外部命令，并将命令标准输出和标准错误输出到指定文件
func RunCommandAndRedirectOut(args []string, stdFile string) (*exec.Cmd, error) {
	if ShowDebugInfo {
		fmt.Printf("run command daemon：%v ，\nout file：%v\n", args, stdFile)
	}
	var cmd *exec.Cmd
	if len(args) == 0 {
		return cmd,fmt.Errorf("执行命令缺少参数！")
	}

	if len(args) > 1 {
		cmd = exec.Command(args[0], args[1:]...)
	} else {
		cmd = exec.Command(args[0])
	}
	if !FileExists(filepath.Dir(stdFile)) {
		os.MkdirAll(filepath.Dir(stdFile), os.ModePerm)
	}
	out,err := os.OpenFile(stdFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil && stdFile != "" {
		return cmd,err
	}

	cmd.Stdin = nil
	cmd.Stdout = out
	cmd.Stderr = out

	err = cmd.Start()
	return cmd, err
}


// 执行命令，并等待命令退出
func RunCommandAndWait(cmdName string, arg ...string) int {
	if ShowDebugInfo {
		fmt.Printf("run command wait%v %v \n", cmdName, arg)
	}
	cmd := exec.Command(cmdName, arg...)
	_ = cmd.Run()
	return cmd.ProcessState.ExitCode()
}
