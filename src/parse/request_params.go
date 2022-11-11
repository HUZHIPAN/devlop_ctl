package parse

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/packer"
	"lwapp/pkg/util"
	"lwapp/src/action"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func ParseRequestPackage(packageFile string) *ApplyHandle {
	packageFilePath := packageFile

	if !filepath.IsAbs(packageFilePath) {
		packageFilePath, err := filepath.Abs(packageFile)
		if err != nil {
			diary.Errorf("获取更新包绝对路径失败: %v, %v", err, packageFilePath)
			return nil
		}
	}

	if !util.FileExists(packageFilePath) {
		diary.Errorf("解析更新包 %v 不存在.", packageFilePath)
		return nil
	}

	unpackPath := GetPackageFileUnpackPath(packageFile)
	os.MkdirAll(unpackPath, os.ModePerm)
	err := packer.NewTgzPacker().UnPack(packageFilePath, unpackPath, []string{}, []string{})

	if err != nil {
		diary.Errorf("解压更新包 %v 失败: %v", packageFilePath, err)
		return nil
	}

	yamlFilePath := unpackPath + "/config.yaml"
	if !util.FileExists(yamlFilePath) {
		diary.Errorf("config.yaml 更新描述文件不存在")
		return nil
	}

	content, err := os.ReadFile(yamlFilePath)
	if err != nil {
		diary.Errorf("解析 config.yaml 失败: %v", err)
		return nil
	}

	requestParams := &action.RequestParams{}
	yaml.Unmarshal(content, requestParams)
	if !verifyRequestParams(unpackPath, requestParams) {
		diary.Errorf("config.yaml 校验失败")
		return nil
	}

	handle := &ApplyHandle{}
	handle.LoadByEventPackages(requestParams.EventPackage)
	handle.SetSourcePath(unpackPath)
	handle.SetRequestParams(*requestParams)
	return handle
}

func GetPackageFileUnpackPath(packageFile string) string {
	return GetUnPackagePath() + "/" + strings.TrimSuffix(filepath.Base(packageFile), ".tar.gz")
}

// 校验更新包参数和更新类型
func verifyRequestParams(unpackPath string, requestParams *action.RequestParams) bool {
	actionType := requestParams.Metadata.ActionType
	if !util.InArray(actionType, []string{"apply"}) {
		diary.Errorf("无效操作 : %v", actionType)
		return false
	}

	// appVersion := requestParams.Metadata.AppVersion
	// if appVersion == "" || len(appVersion) > 100 {
	// diary.Errorf("无效 appVersion : %v", appVersion)
	// return false
	// }
	containType := GetSupportUpdateType()
	eventPackages := requestParams.EventPackage

	if len(eventPackages) == 0 {
		diary.Errorf("更新细节为空 ！")
		// return false
	}

	for _, eventPackage := range eventPackages {
		if !verifyEventPackage(unpackPath, &eventPackage) {
			return false
		}
		containType[eventPackage.Type] += 1
		if containType[eventPackage.Type] > 1 {
			diary.Errorf("多余的更新 : %v", eventPackage.Type)
			return false
		}
	}
	return true
}

func verifyEventPackage(unpackPath string, eventPackage *action.EventPackage) bool {
	name := eventPackage.Name
	if name == "" || len(name) > 255 {
		diary.Errorf("无效的更新包版本号或标识 : %v", name)
		return false
	}
	updateType := eventPackage.Type
	_, exist := GetSupportUpdateType()[updateType]
	if !exist {
		diary.Errorf("无效的更新包类型 : %v", updateType)
		return false
	}
	relativePath := eventPackage.FileRelativePath
	actionPackagePath := unpackPath + "/" + relativePath
	if relativePath == "" || !util.FileExists(actionPackagePath) {
		diary.Errorf("无效的包 : %v", actionPackagePath)
		return false
	}

	return true
}
