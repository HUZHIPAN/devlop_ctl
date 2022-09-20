package common

import (
	"context"
	"lwapp/pkg/diary"
	"lwapp/pkg/docker"
	"lwapp/pkg/gogit"
	"lwapp/pkg/util"
	"os"
	"os/exec"
	"os/user"
	"strconv"
)

var checkChan = make(chan bool, 10)

var UseDefaultUserUid int

func ExecuteBeforeCheckHandle() bool {
	go checkCtlWorkAble()
	go checkDockerStatus()
	go checkEtcPathStatus()
	go checkLwAppPathStatus()

	if os.Getuid() == 0 && !checkDefaultUser() {
		return false
	}

	taskNum := 4
	for {
		ok := <-checkChan
		if !ok {
			return false
		}

		taskNum--
		if taskNum <= 0 {
			return true
		}
	}
}

func checkCtlWorkAble() {
	logPath := GetTmpLogPath()
	if !util.FileExists(logPath) {
		err := os.MkdirAll(logPath, os.ModePerm)
		if err != nil {
			diary.Errorf("无法创建日志目录（%v）：%v", logPath, err)
			checkChan <- false
			return
		}
		diary.Infof("创建工具运行时日志目录（%v）", logPath)
	}

	uploadPath := GetTmpPath() + "/upload_packages"
	if !util.FileExists(uploadPath) {
		err := os.MkdirAll(uploadPath, os.ModePerm)
		if err != nil {
			diary.Errorf("无法创建上传包目录（%v）：%v", uploadPath, err)
			checkChan <- false
			return
		}
		diary.Infof("创建上传包目录（%v）", uploadPath)
	}

	checkChan <- true
}

func checkDockerStatus() {
	dock := docker.NewDockerClient()
	_, err := dock.Ping(context.TODO())
	if err != nil {
		diary.Errorf("无法获取docker环境状态：%v", err)
		checkChan <- false
		return
	}
	// diary.Infof("docker状态正常: %v", s)

	checkChan <- true
}

func checkLwAppPathStatus() {
	lwappPath := GetLwappPath()
	if !util.FileExists(lwappPath) {
		err := os.MkdirAll(lwappPath, os.ModePerm)
		if err != nil {
			diary.Errorf("创建目录（%v）失败：%v", lwappPath, err)
			checkChan <- false
			return
		} else {
			diary.Infof("创建目录（%v）成功", lwappPath)
		}
	}

	if !gogit.IsRepository(lwappPath) {
		if !gogit.InitializeDirVersionControl(lwappPath) {
			diary.Errorf("无法初始化目录（%v）版本控制", lwappPath)
			checkChan <- false
			return
		} else {
			diary.Infof("初始化目录版本控制（%v）成功", lwappPath)
		}
	}

	if !gogit.IsDirVersionInitialized(lwappPath) {
		if !initCommitRepository(lwappPath) {
			diary.Errorf("初始化仓库提交失败:%v", lwappPath)
			checkChan <- false
			return
		} else {
			diary.Infof("初始化仓库提交（%v）成功", lwappPath)
		}
	}

	checkChan <- true
}

func initCommitRepository(dirPath string) bool {
	return gogit.CommitDirChange(dirPath, "初始化", GetLwappIgnoreExpression())
}

func checkEtcPathStatus() {
	webServerlogPath := GetDeploymentLogPath()

	if !util.FileExists(webServerlogPath) {
		err := os.MkdirAll(webServerlogPath, os.ModePerm)
		if err != nil {
			diary.Errorf("创建webServer日志目录（%v）失败：%v", webServerlogPath, err)
			checkChan <- false
			return
		} else {
			diary.Infof("创建webServer日志目录（%v）成功", webServerlogPath)
		}
	}

	if !util.FileExists(webServerlogPath + "/exec") {
		err := os.MkdirAll(webServerlogPath+"/exec", os.ModePerm)
		if err != nil {
			diary.Errorf("无法创建容器exec日志目录（%v）：%v", webServerlogPath+"/exec", err)
			checkChan <- false
			return
		} else {
			diary.Infof("创建容器exec日志目录（%v）成功", webServerlogPath+"/exec")
		}
	}

	if !util.FileExists(webServerlogPath + "/nginx") {
		err := os.MkdirAll(webServerlogPath+"/nginx", os.ModePerm)
		if err != nil {
			diary.Errorf("无法创建容器nginx日志目录（%v）：%v", webServerlogPath+"/nginx", err)
			checkChan <- false
			return
		} else {
			diary.Infof("创建容器nginx日志目录（%v）成功", webServerlogPath+"/nginx")
		}
	}

	etcPath := GetEtcPath()
	if !util.FileExists(etcPath) {
		err := os.MkdirAll(etcPath, os.ModePerm)
		if err != nil {
			diary.Errorf("创建目录（%v）失败：%v", etcPath, err)
			checkChan <- false
			return
		} else {
			diary.Infof("创建目录（%v）成功", etcPath)
		}
	}

	if !gogit.IsRepository(etcPath) {
		if !gogit.InitializeDirVersionControl(etcPath) {
			diary.Warningf("无法初始化目录（%v）版本控制", etcPath)
			checkChan <- false
			return
		} else {
			diary.Infof("初始化目录版本控制（%v）成功", etcPath)
		}

	}

	if !gogit.IsDirVersionInitialized(etcPath) {
		if !initCommitRepository(etcPath) {
			diary.Errorf("初始化仓库提交失败:%v", etcPath)
			checkChan <- false
			return
		} else {
			diary.Infof("初始化提交（%v）成功", etcPath)
		}
	}

	checkChan <- true
}

// 检查默认使用的用户
func checkDefaultUser() bool {
	u, err := user.Lookup(DefaultUser)
	if err != nil {
		useradd := exec.Command("useradd", DefaultUser)
		err := useradd.Start()
		if err != nil {
			diary.Errorf("执行创建%v用户命令失败：%v", DefaultUser, err)
			return false
		}
		err = useradd.Wait()
		if err != nil || useradd.ProcessState.ExitCode() != 0 {
			diary.Errorf("创建%v用户失败：%v, exit code：%v", DefaultUser, err, useradd.ProcessState.ExitCode())
			return false
		}
		created_u, err := user.Lookup(DefaultUser)
		if err != nil {
			diary.Errorf("执行创建用户操作完成后任未找到用户（%v）：%v", DefaultUser, err)
			return false
		}
		diary.Infof("创建系统用户（%v）成功，uid：%v", DefaultUser, created_u.Uid)
		UseDefaultUserUid, err = strconv.Atoi(created_u.Uid)
		if UseDefaultUserUid <= 0 || err != nil {
			diary.Errorf("获取默认用户uid失败：%v", err)
			return false
		}
		return true
	}

	UseDefaultUserUid, _ = strconv.Atoi(u.Uid)
	if UseDefaultUserUid <= 0 || err != nil {
		diary.Errorf("获取默认用户uid失败：转换出错：%v", err)
		return false
	}
	return true
}

// 改变目录所有者权限
func ChownDirectoryPower(dirPath string) {
	if UseDefaultUserUid <= 0 {
		return
	}
	err := util.ChownAll(dirPath, UseDefaultUserUid, -1)
	if err != nil {
		diary.Warningf("设置目录（%v）所有者为（%v），uid：（%v）发生异常：%v！", dirPath, DefaultUser, UseDefaultUserUid, err.Error())
	} else {
		diary.Infof("设置目录（%v）所有者为（%v）成功，uid：（%v）", dirPath, DefaultUser, UseDefaultUserUid)
	}
}
