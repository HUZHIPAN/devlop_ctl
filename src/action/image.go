package action

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"os"
	"strconv"
	"strings"
	"time"
)

type ImageApplyResult struct {
	IsSuccess bool
	ExistDiff bool
}

var WebRunContainerName = "lwops_web"

func ImageUpdateApply(sourcePath string, event *EventPackage) *ImageApplyResult {
	result := &ImageApplyResult{
		IsSuccess: false,
	}

	currentVersionNumber := GetLastWebImageTagNumber()
	if currentVersionNumber == -1 {
		return result
	}

	rootfsPackageFile := sourcePath + "/" + event.FileRelativePath
	rootfsPath := common.GetRootfsPath()

	imgTag := event.Name // 更新包镜像tag
	refreshAfterNumber := currentVersionNumber + 1

	if util.FileExists(fmt.Sprintf("%v/%d/%v", rootfsPath, currentVersionNumber, imgTag)) {
		diary.Infof("加载镜像（%v），与当前最新镜像版本一致，已略过更新", imgTag)
		result.ExistDiff = false
	} else {
		refreshPath := fmt.Sprintf("%v/%d", rootfsPath, refreshAfterNumber)
		os.MkdirAll(refreshPath, os.ModePerm)
		exitCode := util.RunCommandAndWait("tar", "-xvf", rootfsPackageFile, "-C", refreshPath)
		if exitCode == 0 {
			util.WriteFileWithDir(fmt.Sprintf("%v/%d/%v", rootfsPath, refreshAfterNumber, imgTag), imgTag)
			diary.Infof("更新镜像roofs成功，更新最新版本（%v），更新ID（%v）", imgTag, refreshAfterNumber)
			result.ExistDiff = true
		} else {
			diary.Errorf("更新镜像失败，执行tar命令解压发生错误，exitCode：%v", exitCode)
			os.RemoveAll(refreshPath)
			return result
		}
	}

	result.IsSuccess = true
	diary.Infof("更新镜像包成功")
	return result
}

// 获取镜像（rootfs）最新版本号
func GetLastWebImageTagNumber() int {
	var currentMaxVersionNumber int64 = 0
	rootfsPath := common.GetRootfsPath()
	fileList := util.GetDirFileList(rootfsPath)

	for _, dirInfo := range fileList {
		if !dirInfo.IsDir() {
			continue
		}
		number, err := strconv.ParseInt(dirInfo.Name(), 0, 64)
		if err != nil {
			continue
		}
		if number > currentMaxVersionNumber {
			currentMaxVersionNumber = number
		}
	}

	if currentMaxVersionNumber == 0 {
		diary.Errorf("未加载过基础运行镜像！")
	}

	return int(currentMaxVersionNumber)
}

// 执行web容器内的命令，工作目录（/itops/nginx/html/lwjk_app）
func RunContainerCommand(runCmd string, interactive bool, delay int) bool {
	webContainer := GetCurrentRunningWebContainer()
	if webContainer == nil {
		return false
	}

	logFileName := time.Now().Format("2006-01-02") + "_command_log" + ".txt"
	execLogFile := "/itops/logs/exec/" + logFileName
	execLogMessage := fmt.Sprintf("\n%v 执行命令（%v）：>>>>>>>>>>>>>>>>>>>>>>>>>>>>", time.Now().Format("2006-01-02 15:04:05"), runCmd)
	runCommandUser := fmt.Sprintf("%d", common.GetDeployEnvParams().Uid)
	lwjkAppPath := "/itops/nginx/html/lwjk_app"

	if interactive {
		util.RunCommandWithCli(GetRuncBin(), "--root", GetRuncStatePath(), "exec", "--user", runCommandUser, "--cwd", lwjkAppPath, "-t", webContainer.Name, "sh", "-c", runCmd)
		return true
	} else {
		execCommand := fmt.Sprintf("sleep %d; echo '%v' >> %v; %v &>> %v", delay, execLogMessage, execLogFile, runCmd, execLogFile)
		exitCode := util.RunCommandAndWait(GetRuncBin(), "--root", GetRuncStatePath(), "exec", "--user", runCommandUser, "--cwd", lwjkAppPath, "-d", webContainer.Name, "sh", "-c", execCommand)
		if exitCode != 0 {
			diary.Errorf("执行容器内命令（%v）发生错误，exitCode：%v", runCmd, exitCode)
			return false
		} else {
			defer diary.Infof("发送exec命令（%v）成功，可在（%v）中查看执行结果", runCmd, common.GetWebExecLogPath()+"/"+logFileName)
			return true
		}
	}
}

// 执行容器内脚本（非交互式）
func RunContainerScript(commandFile string, delay int) bool {
	return RunContainerCommand(fmt.Sprintf("sh -c '%v'", commandFile), false, delay)
}

// 启动web容器
func RunContainer() bool {
	webContainer := GetCurrentRunningWebContainer()
	if webContainer != nil {
		diary.Errorf("WEB容器已经已经在运行中！")
		return false
	}

	if !strings.Contains(gogit.GetRepositoryCurrentBranch(common.GetEtcPath()), EtcBranchRuntimeSuffix) {
		diary.Errorf("未生成运行配置，请先使用 lwctl build 命令创建运行环境配置！")
		return false
	}

	runcStateFile := fmt.Sprintf("%v/%v", GetRuncStatePath(), getRuncProcessContainerName())
	if util.FileExists(runcStateFile) { // 容器停止，但其状态文件未删除，通常是异常退出导致
		os.RemoveAll(runcStateFile)
	}

	CheckAndCreatePersistenceDir() // 检查软链和持久化目录

	nginxStartStdoutFile := common.GetDeploymentLogPath() + "/nginx/nginx_stdout.log"
	runcStartStdoutFile := common.GetTmpRuncPath() + "/start.log"
	os.Remove(nginxStartStdoutFile)
	os.Remove(runcStartStdoutFile)

	startUpCommand := []string{GetRuncBin(), "--root", GetRuncStatePath(), "run", "--pid-file", getContainerStartupProcessPidFile(), getRuncProcessContainerName(), "--bundle", common.GetEtcRuncPath()}
	runcCmd, err := util.RunCommandAndRedirectOut(startUpCommand, runcStartStdoutFile)
	if err != nil {
		diary.Infof("启动失败：执行命令（%v）失败：%v", startUpCommand, err)
		return false
	}

	beginStart := time.Now().Unix()
	for {
		// 轮询检查nginx启动输出
		// 存在输出文件 或 10s超时 结束
		if util.FileExists(nginxStartStdoutFile) {
			time.Sleep(time.Millisecond * 300)
			break
		}
		time.Sleep(time.Millisecond * 100)
		if time.Now().Unix()-beginStart >= 10 {
			break
		}
	}

	util.RunCommandWithCli("reset", "-c", "xterm")

	diary.Infof("启动命令：\n%v", startUpCommand)

	runcOut, err := os.ReadFile(runcStartStdoutFile)
	if err != nil {
		diary.Infof("查看runc进程标准输出和错误输出异常：%v", err)
	} else {
		if string(runcOut) != "" {
			diary.Infof("runc启动进程输出日志：\n%v", string(runcOut))
		}
	}

	out, err := os.ReadFile(nginxStartStdoutFile)
	if err != nil {
		diary.Errorf("WEB容器启动失败，没有nginx进程输出日志：%v", err)
		return false
	}

	if string(out) != "" {
		diary.Warningf("容器nginx启动输出：\n%v", string(out))
	}

	if !util.CheckProcessIsRunning(runcCmd.Process.Pid, "runc") {
		diary.Errorf("WEB容器启动失败：runc进程（%v）已退出！", runcCmd.Process.Pid)
		return false
	}

	if !util.CheckProcessIsRunning(getContainerStartupProcessRunningPid(), "sh") {
		diary.Errorf("WEB容器启动失败：启动脚本（start.sh）已退出！")
		return false
	}

	_, err = util.WriteFileWithDir(getContainerRuncPidFile(), fmt.Sprintf("%d", runcCmd.Process.Pid))
	if err != nil {
		diary.Errorf("启动容器写入runc进程pid文件（%v）失败：%v", getContainerRuncPidFile(), err)
		return false
	}

	diary.Infof("容器启动成功，runc进程pid：%v ", runcCmd.Process.Pid)
	return true
}

// 获取启动容器runc进程的pid文件
func getContainerRuncPidFile() string {
	return common.GetTmpRuncPath() + "/runc_container.pid"
}

// 获取容器启动进程的pid文件
func getContainerStartupProcessPidFile() string {
	return common.GetTmpRuncPath() + "/container_startup.pid"
}

// 获取当前部署目录唯一标识
func GetCurrentPathEnvContainerSuffix() string {
	currentEnvPath := common.GetLwopsVolume()
	suffix := strings.ReplaceAll(currentEnvPath, "/", "_")
	suffix = strings.ReplaceAll(suffix, "\\", "_")
	suffix = strings.ReplaceAll(suffix, ":", "_")
	return suffix
}

// 获取当前存在并运行的容器
func GetCurrentRunningWebContainer() *Container {
	runcProcessIsRunning := checkRuncProcessIsRunning()
	containerProcessIsRunning := checkContainerProcessIsRunning()

	// diary.Debugf("runc :%v , container: %v ", runcProcessIsRunning, containerProcessIsRunning)
	if !runcProcessIsRunning && containerProcessIsRunning { // runc进程退出，容器进程未结束
		exitCode := forceKillContainerProcess()
		diary.Errorf("runc进程已退出，但容器内尚有进程在运行中，发送强制关闭信号，exitCode：%v", exitCode)
		return nil
	}

	if runcProcessIsRunning && containerProcessIsRunning {
		return &Container{
			Pid:  getRuncProcessRunningPid(),
			Name: getRuncProcessContainerName(),
		}
	}
	return nil
}

// 检测runc进程是否在运行中
func checkRuncProcessIsRunning() bool {
	return util.CheckProcessIsRunning(getRuncProcessRunningPid(), "runc")
}

// 获取runc进程pid
func getRuncProcessRunningPid() int {
	if !util.FileExists(getContainerRuncPidFile()) {
		return -1
	}
	pidContent, err := os.ReadFile(getContainerRuncPidFile())
	if err != nil {
		return -1
	}
	pidStr := string(pidContent)
	pidStr = strings.ReplaceAll(pidStr, "\n", "")
	runcPid, err := strconv.Atoi(pidStr)
	if err != nil {
		return -1
	}
	return runcPid
}

// 获取容器启动进程（pid为1）在全局命名空间的进程pid
func getContainerStartupProcessRunningPid() int {
	pidContent, err := os.ReadFile(getContainerStartupProcessPidFile())
	if err != nil {
		return -1
	}
	pidStr := string(pidContent)
	pidStr = strings.ReplaceAll(pidStr, "\n", "")
	containerStartupProcessPid, err := strconv.Atoi(pidStr)
	if err != nil {
		return -1
	}
	return containerStartupProcessPid
}

// 检测容器中是否有进程在运行中
func checkContainerProcessIsRunning() bool {
	exitCode := util.RunCommandAndWait(GetRuncBin(), "--root", GetRuncStatePath(), "kill", getRuncProcessContainerName(), "0")
	return exitCode == 0
}

// runc可执行文件
func GetRuncBin() string {
	return common.GetEtcRuncPath() + "/runc"
}

// runc进程状态保存目录
func GetRuncStatePath() string {
	return common.GetTmpRuncPath() + "/state"
}

// 获取当前部署目录启动容器标识
func getRuncProcessContainerName() string {
	return WebRunContainerName + GetCurrentPathEnvContainerSuffix()
}

// 发送强制退出信号到容器内进程
func forceKillContainerProcess() int {
	exitCode := util.RunCommandAndWait(GetRuncBin(), "--root", GetRuncStatePath(), "kill", "--all", getRuncProcessContainerName(), "SIGKILL")
	if exitCode != 0 {
		// 当命名空间内pid为1的进程退出后，内核会发送SIGKILL信号到该命名空间下所有进程
		exitCode = util.RunCommandAndWait("kill", "-9", fmt.Sprintf("%d", getContainerStartupProcessRunningPid()))
	}
	return exitCode
}

// 停止启动的web容器
func StopContainer() bool {
	webContainer := GetCurrentRunningWebContainer()
	if webContainer == nil {
		diary.Errorf("没有容器在运行中！")
		return false
	}

	exitCode := util.RunCommandWithCli(GetRuncBin(), "--root", GetRuncStatePath(), "kill", "--all", webContainer.Name, "SIGTERM")
	if exitCode != 0 {
		diary.Errorf("向runc进程发送SIGTERM信号失败，exitCode：%v ", exitCode)
		return false
	}

	beginStart := time.Now().Unix()
	for {
		if time.Now().Unix()-beginStart >= 10 {
			forceKillContainerProcess()
			diary.Warningf("关闭容器（%v）超时，发送强制退出SIGKILL信号！\n", webContainer.Name)
			time.Sleep(time.Millisecond * 200)
			break
		}

		if util.CheckProcessIsRunning(webContainer.Pid, "runc") || checkContainerProcessIsRunning() {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		break
	}

	diary.Infof("关闭容器（%v）成功", webContainer.Name)
	return true
}

// 获取部署目录使用的用户权限uid
func GetUseRunContainerUserUid() int {
	useUid := os.Getuid()
	if useUid == 0 {
		useUid = common.UseDefaultUserUid // 当前系统itops用户，uid不定
	}
	return useUid
}
