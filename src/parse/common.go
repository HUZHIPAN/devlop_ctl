package parse

import "lwapp/src/common"

func GetSupportUpdateType() map[string]int {
	return map[string]int{
		"product":   0,
		"feature":   0,
		"customer":  0,
		"configure": 0,
		"image":     0,
	}
	// return []string{"product", "feature", "customer", "configure", "image"}
}

func GetRequestPackagePath() string {
	return common.GetTmpPath() + "/upload_packages"
}

func GetUnPackagePath() string {
	return common.GetTmpPath() + "/unpack"
}
