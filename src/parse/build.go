package parse

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"lwapp/src/structure"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// 获取部署 .env文件路径
func getDeployEnvFilePath() string {
	return common.GetEnvironmentVolume() + "/.env"
}

// 是否存在环境配置
func IsExistDeployEnvSetting() bool {
	return util.FileExists(getDeployEnvFilePath())
}

// 判断是否使用默认build参数
func IsDefaultBuildParams(params *structure.BuildParams) bool {
	return params.WebPort == common.DefaultWebPort && params.WebApiPort == common.DefaultWebApiPort && params.MacAddr == "" && params.WebApiGateway == "/backend_api"
}

// 获取部署环境配置相关信息
func GetDeployEnvParams() *structure.EnvironmentParams {
	envFile := getDeployEnvFilePath()
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

// 生成环境配置信息
func GenerateEnvBuildParams(params *structure.BuildParams, uid int) {
	envFile := common.GetEnvironmentVolume() + "/.env"
	var env *structure.EnvironmentParams
	if !util.FileExists(envFile) {
		env = &structure.EnvironmentParams{}
	} else {
		env = GetDeployEnvParams()
	}

	env.Build.BuildTime = time.Now().Format("2006-01-02 15:04:05")
	env.Build.WebPort = params.WebPort
	env.Build.WebApiPort = params.WebApiPort
	env.Build.WebApiGateWay = params.WebApiGateway
	env.Build.MacAddr = params.MacAddr

	env.Uid = uid

	out, err := yaml.Marshal(env)
	if err != nil {
		diary.Errorf("生成环境配置（yaml.Marshal(env)）错误：%v，配置：", err, env)
		return
	}

	_, err = util.WriteFileWithDir(envFile, string(out))
	if err != nil {
		diary.Warningf("写入环境配置文件（%v）错误：%v", envFile, err)
	} else {
		diary.Infof("重新生成环境配置文件：%v，配置：%v", envFile, env)
	}
}

// 通过部署环境配置重新生成容器
func BuildContainerByEnvParams() bool {
	envParams := GetDeployEnvParams()
	buildParams := &structure.BuildParams{
		WebPort:       envParams.Build.WebPort,
		WebApiPort:    envParams.Build.WebApiPort,
		WebApiGateway: envParams.Build.WebApiGateWay,
		MacAddr:       envParams.Build.MacAddr,
	}

	if BuildCommandHandle(buildParams) {
		diary.Infof("根据部署环境配置：%v，重新生成WEB容器成功", envParams.Build)
		return true
	} else {
		diary.Errorf("生成WEB容器失败，环境配置：%v", envParams.Build)
		return false
	}
}
