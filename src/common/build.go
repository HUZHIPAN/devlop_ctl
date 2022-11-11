package common

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"lwapp/src/structure"
	"os"

	"gopkg.in/yaml.v2"
)

// 获取部署 .env文件路径
func GetDeployEnvFilePath() string {
	return GetDeploymentVolume() + "/.env"
}

// 是否存在环境配置
func IsExistDeployEnvSetting() bool {
	return util.FileExists(GetDeployEnvFilePath())
}

// 判断是否使用默认build参数
func IsDefaultBuildParams(params *structure.BuildParams) bool {
	return params.WebPort == DefaultWebPort && params.WebApiPort == DefaultWebApiPort && params.WebApiGateway == "/backend_api"
}

// 获取部署环境配置相关信息
func GetDeployEnvParams() *structure.EnvironmentParams {
	envFile := GetDeployEnvFilePath()
	envContent, err := os.ReadFile(envFile)
	if err != nil {
		diary.Warningf("读取（%v）文件的环境配置失败：%v", err)
	}
	env := &structure.EnvironmentParams{}
	err = yaml.Unmarshal(envContent, env)
	if err != nil {
		diary.Warningf("解析（%v）文件的环境配置发生错误：%v", err)
	}
	return env
}

// 启动php-fmp监听的unix socket文件
func GetPhpfpmSocketListenFile() string {
	return GetLwopsVolume() + "/phpfpm.sock"
}
