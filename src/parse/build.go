package parse

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"lwapp/src/structure"
	"time"

	"gopkg.in/yaml.v2"
)

// 生成环境配置信息
func GenerateEnvBuildParams(params *structure.BuildParams, uid int) {
	envFile := common.GetDeployEnvFilePath()
	var env *structure.EnvironmentParams
	if !util.FileExists(envFile) {
		env = &structure.EnvironmentParams{}
	} else {
		env = common.GetDeployEnvParams()
	}

	env.Build.BuildTime = time.Now().Format("2006-01-02 15:04:05")
	env.Build.WebPort = params.WebPort
	env.Build.WebApiPort = params.WebApiPort
	env.Build.WebApiGateWay = params.WebApiGateway

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
	envParams := common.GetDeployEnvParams()
	buildParams := &structure.BuildParams{
		WebPort:       envParams.Build.WebPort,
		WebApiPort:    envParams.Build.WebApiPort,
		WebApiGateway: envParams.Build.WebApiGateWay,
	}

	if BuildCommandHandle(buildParams) {
		diary.Infof("根据部署环境配置：%v，重新生成WEB运行配置成功", envParams.Build)
		return true
	} else {
		diary.Errorf("生成WEB运行配置失败，环境配置：%v", envParams.Build)
		return false
	}
}
