package action

import (
	"lwapp/src/common"
	"strings"
)

type RequestParams struct {
	Metadata struct {
		ActionType  string   `yaml:"actionType"`
		AppVersion  string   `yaml:"appVersion"`
		Description string   `yaml:"description"`
		Commands    []string `yaml:"commands"`
	} `yaml:"metadata"`

	// ReloadContainer bool
	EventPackage []EventPackage `yaml:"components"`
}

type EventPackage struct {
	Name             string   `yaml:"name"`
	Type             string   `yaml:"type"`
	Description      string   `yaml:"desc"`
	FileRelativePath string   `yaml:"package"`
	Commands         []string `yaml:"commands"`
}

// 根据分支名称获取产品更新版本号
func GetAppVersionByBranchName(branchName string) string {
	appVersion := strings.TrimPrefix(branchName, common.FullPackagePrefix)
	appVersion = strings.TrimPrefix(appVersion, common.IncrPackagePrefix)
	return appVersion
}

type Container struct {
	Pid int
	Name string
}