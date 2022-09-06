package common

import "strings"

var (
	FullPackagePrefix   = "full_"   // 全量包分支前缀
	IncrPackagePrefix   = "incr_"   // 增量包分支前缀
	CustomPackagePrefix = "custom_" // 自定义类型
)

type AppVersionMode struct {
	Mode          string
	VersionNumber string
}

func GetAppVersionNumberInfo(appVersion string) AppVersionMode {
	versionMode := AppVersionMode{}
	if strings.Contains(appVersion, FullPackagePrefix) {
		versionMode.Mode = "全量"
		versionMode.VersionNumber = strings.TrimPrefix(appVersion, FullPackagePrefix)
		return versionMode
	}

	if strings.Contains(appVersion, IncrPackagePrefix) {
		versionMode.Mode = "增量"
		versionMode.VersionNumber = strings.TrimPrefix(appVersion, IncrPackagePrefix)
		return versionMode
	}

	versionMode.VersionNumber = appVersion
	return versionMode
}
