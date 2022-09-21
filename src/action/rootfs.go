package action

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"os"
)

func RootfsUpdateApply(sourcePath string, event *EventPackage) bool {
	rootfsPackageFile := sourcePath+"/"+event.FileRelativePath
	rootfsPath := common.GetRootfsPath()
	if util.FileExists(rootfsPath) {
		exitCode := util.RunCommandAndWait("mv", rootfsPath, rootfsPath+".bak")
		if exitCode != 0 {
			diary.Errorf("执行mv命令，更新rootfs前备份失败！")
			return false
		}
	}

	exitCode := util.RunCommandAndWait("tar", "-xvf", rootfsPackageFile, "-C", common.GetRootfsPath())
	if exitCode == 0 {
		os.RemoveAll(rootfsPath+".bak")
		return true
	} else {
		os.RemoveAll(rootfsPath)
		exitCode = util.RunCommandAndWait("mv", rootfsPath+".bak", rootfsPath)
		if exitCode != 0 {
			diary.Errorf("执行mv命令，回滚备份的rootfs失败！")
			return false
		} else {
			return true
		}
	}
}