package common

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"os"
	"path/filepath"
)

var (
	lwopsPath string
)

var (
	DefaultLwopsPath  string = "/lwops" // 默认部署目录
	DefaultWebPort    int    = 80	// 默认WEB前端访问端口
	DefaultWebApiPort int    = 8081 // 默认WEB后端访问端口

	DefaultUser string = "itops"  // 当使用root用户部署时，默认使用的用户权限（不存在会创建）
)

// 忽略目录
var LwappIgnoreDirectoryExpression = []string{
	"/environments",
	"/runtime",
	"/web/config",
	"/web/uploads",
	"/web/z",
	"/web/zbx",
	"/web/assets",
}

// 忽略文件
var LwappIgnoreFileExpression = []string{
	"/.env",     // 环境配置
	"/.license", // 授权码

	"/web/app/config.json", // 前端资源配置文件
}

// 设置部署目录
func SetLwopsVolume(opsPath string) bool {
	if !util.FileExists(opsPath) {
		err := os.MkdirAll(opsPath, os.ModePerm)
		if err != nil {
			fmt.Println("创建部署目录发生错误：", err)
			return false
		} else {
			fmt.Printf("创建部署目录（%v）成功\n", opsPath)
		}
	}
	opsPathAbsolute, err := filepath.Abs(opsPath)
	if err != nil {
		fmt.Printf("获取部署目录（%v）绝对路径错误：%v\n", opsPath, err)
		return false
	}
	lwopsPath = opsPathAbsolute

	if !LockLwopsEnv() {
		fmt.Println("无法对部署目录加锁，可能其他进程正在操作，请检查！")
		return false
	}

	diary.SetLogPath(GetTmpLogPath())

	diary.Infof("部署目录：%v", lwopsPath)
	return true
}

// 获取部署的根目录
func GetLwopsVolume() string {
	return lwopsPath
}

// 获取持久化数据目录
func GetPersistenceVolume() string {
	return GetLwopsVolume() + "/data"
}

// 环境/代码 目录
func GetEnvironmentVolume() string {
	return GetLwopsVolume() + "/deployment"
}

// 部署工具运行时目录
func GetTmpPath() string {
	return GetLwopsVolume() + "/tmp"
}

// 部署工具运行时日志目录
func GetTmpLogPath() string {
	return GetTmpPath() + "/logs"
}

// 运行环境配置目录
func GetEtcPath() string {
	return GetEnvironmentVolume() + "/etc"
}

func GetDeploymentLogPath() string {
	return GetEnvironmentVolume() + "/logs"
}

// 获取web容器exec执行日志目录
func GetWebExecLogPath() string {
	return GetDeploymentLogPath() + "/exec"
}

// web项目代码主目录
func GetLwappPath() string {
	return GetEnvironmentVolume() + "/lwjk_app"
}

// 获取lwjk_app忽略管控的文件和目录
func GetLwappIgnoreExpression() []string {
	return append(LwappIgnoreDirectoryExpression, LwappIgnoreFileExpression...)
}
