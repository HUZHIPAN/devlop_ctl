package action

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/pkg/packer"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"os"
	"path/filepath"
)

type ProductApplyResult struct {
	IsSuccess     bool
	ExistDiff     bool
	LwAppPath     string
	BeforeVersion string
	AfterVersion  string
}

func ProductUpdateApply(sourcePath string, event *EventPackage) *ProductApplyResult {
	result := &ProductApplyResult{
		IsSuccess: false,
	}

	lwAppPath := common.GetLwappPath()
	currentBranchName := gogit.GetRepositoryCurrentBranch(lwAppPath)
	if currentBranchName == "" {
		diary.Errorf("仓库（%v）无法获取仓库当前分支！", lwAppPath)
		result.IsSuccess = false
		return result
	}

	// 分支命名
	appVersion := common.FullPackagePrefix + event.Name

	result.BeforeVersion = currentBranchName
	result.AfterVersion = appVersion

	// 更新版本与当前环境相同版本
	if appVersion == currentBranchName {
		result.ExistDiff = false
		result.IsSuccess = true
		diary.Infof("更新包版本（%v）与当前版本相同，已略过产品包更新", appVersion)
		return result
	}

	if !gogit.CheckBranchIsExist(lwAppPath, appVersion) {
		if gogit.CreateRepositoryBranchFromByMaster(lwAppPath, appVersion) {
			diary.Infof("仓库（%v）创建版本分支（%v）成功", lwAppPath, appVersion)
		} else {
			diary.Errorf("仓库（%v）无法为新版本创建分支：%v", lwAppPath, appVersion)
			return result
		}
	} else {
		diary.Warningf("仓库（%v）已经存在版本分支（%v）！", lwAppPath, appVersion)
	}

	isExistRunContainer := GetCurrentRunningWebContainer() != nil
	if isExistRunContainer {
		// 开始全量更新前，先停止容器
		diary.Infof("当前部署目录（%v）存在运行中的容器", common.GetLwopsVolume())
		StopContainer()
	}

	if !gogit.CheckoutRepositoryBranch(lwAppPath, appVersion, common.GetLwappIgnoreExpression()) {
		diary.Errorf("仓库（%v）切换至新版本分支失败：%v", lwAppPath, appVersion)
		RollbackProductOnFail(currentBranchName, isExistRunContainer)
		return result
	} else {
		diary.Infof("仓库（%v）切换至新版本分支（%v）", lwAppPath, appVersion)
	}

	ignorePrefix := util.Array_merge([]string{".git/"}, common.GetLwappIgnoreExpression())
	err := packer.NewTgzPacker().UnPack(sourcePath+"/"+event.FileRelativePath, lwAppPath, ignorePrefix, []string{".gitignore"})
	if err != nil {
		diary.Errorf("更新全量产品包操作，解压至（%v）目录时发生异常: %v", lwAppPath, err)
		RollbackProductOnFail(currentBranchName, isExistRunContainer)
		return result
	} else {
		diary.Infof("更新全量产品包解压至目录（%v）成功", lwAppPath)
	}

	statusList, err := gogit.RepositoryWorkSpaceStatus(lwAppPath, common.GetLwappIgnoreExpression())
	if err != nil {
		diary.Errorf("仓库（%v）获取工作区变动发生异常: %v", lwAppPath, err)
		RollbackProductOnFail(currentBranchName, isExistRunContainer)
		return result
	}
	commitComment := fmt.Sprintf("产品包更新：更新前版本（%v），更新版本（%v），更新描述：%v", currentBranchName, event.Name, event.Description)
	if !gogit.CommitDirChange(lwAppPath, commitComment, common.GetLwappIgnoreExpression()) {
		diary.Errorf("仓库（%v）提交新版本代码失败！%v个文件变动！", lwAppPath, len(statusList))
		RollbackProductOnFail(currentBranchName, isExistRunContainer)
		return result
	} else {
		diary.Infof("仓库（%v）提交变动成功:%v 个文件变动", lwAppPath, len(statusList))
	}

	result.ExistDiff = true
	result.IsSuccess = true
	result.LwAppPath = lwAppPath

	diary.Infof("更新产品更新包成功")

	// 检查产品目录的软链及目录权限
	CheckAndCreatePersistenceDir()
	common.ChownDirectoryPower(common.GetLwappPath())

	if isExistRunContainer {
		// 全量更新完成，尝试启动容器
		RunContainer()
		RunAppInitializationCommand()
	}

	return result
}

// 丢弃当前变动，切换至某个版本分支
func RollbackProductOnFail(version string, isStart bool) {
	diary.Warningf("尝试回滚产品版本至（%v），更新操作前容器是否启动：%v", isStart)
	gogit.CleanRepositoryWorkspaceChange(common.GetLwappPath(), common.GetLwappIgnoreExpression()) // 回滚未提交变动
	common.ChownDirectoryPower(common.GetLwappPath())
	if CheckoutAppVersion(version) {
		if isStart {
			RunContainer()
			RunAppInitializationCommand()
		}
	}
}

func CheckAndCommitLwappChange() int {
	lwappPath := common.GetLwappPath()
	workTreeStatus, err := gogit.RepositoryWorkSpaceStatus(lwappPath, common.GetLwappIgnoreExpression())
	if err != nil {
		return 0
	}
	if len(workTreeStatus) > 0 {
		diary.Warningf("仓库（%v）不是干净的，这表示程序运行期间可能存在修改忽略目录之外的文件%v个", lwappPath, len(workTreeStatus))
		if !gogit.CommitDirChange(lwappPath, "来源于产品更新包之外的改动，更新前自动提交", common.GetLwappIgnoreExpression()) {
			diary.Errorf("lwjk_app项目代码更新前自动提交失败：\n%v", workTreeStatus)
			return len(workTreeStatus)
		} else {
			diary.Infof("lwjk_app项目代码更新前自动提交成功：\n%v", workTreeStatus)
		}
	}

	return 0
}

// 检查创建持久化目录
func CheckAndCreatePersistenceDir() bool {
	lwappDataPath := common.GetPersistenceVolume()
	lwappPath := common.GetLwappPath()

	for _, filenamePath := range common.LwappIgnoreDirectoryExpression {
		oldName, _ := filepath.Abs(lwappDataPath + filenamePath)
		newName, _ := filepath.Abs(lwappPath + filenamePath)

		if !util.FileExists(oldName) {
			err := os.MkdirAll(oldName, os.ModePerm)
			if err != nil {
				diary.Errorf("无法创建持久化目录（%v）: %v", oldName, err)
			} else {
				diary.Infof("创建持久化目录（%v）", oldName)
			}
		}
		_ = os.MkdirAll(filepath.Dir(newName), os.ModePerm)

		// 打开出错，或者不是软链接
		_, err := os.Readlink(newName)
		if err != nil {
			os.RemoveAll(newName)
			err = os.Symlink(oldName, newName)
			if err != nil {
				diary.Errorf("无法创建目录符号连接（%v -> %v）：%v", newName, oldName, err)
			} else {
				diary.Infof("创建目录符号连接（%v -> %v）成功", newName, oldName)
			}
		}
	}

	for _, filenamePath := range common.LwappIgnoreFileExpression {
		oldName, _ := filepath.Abs(lwappDataPath + filenamePath)
		newName, _ := filepath.Abs(lwappPath + filenamePath)

		_ = os.MkdirAll(filepath.Dir(oldName), os.ModePerm)
		_ = os.MkdirAll(filepath.Dir(newName), os.ModePerm)

		// 不是软链接
		_, err := os.Readlink(newName)
		if err != nil {
			err = os.Symlink(oldName, newName)
			if err != nil {
				diary.Errorf("无法创建文件符号连接（%v -> %v）：%v", newName, oldName, err)
			} else {
				diary.Errorf("创建文件符号连接（%v -> %v）成功", newName, oldName)
			}
		}
	}

	os.MkdirAll(lwappPath+"/web/assets", os.ModePerm)
	return true
}

// 切换版本号
func CheckoutAppVersion(appVersion string) bool {
	lwappPath := common.GetLwappPath()
	ok := gogit.CheckoutRepositoryBranch(lwappPath, appVersion, common.GetLwappIgnoreExpression())
	if !ok {
		diary.Errorf("尝试切换至版本（%v）失败：%v", appVersion, lwappPath)
		return false
	} else {
		diary.Infof("切换至（%v）成功", appVersion)
	}

	return true
}

// 执行web初始化命令
func RunAppInitializationCommand() bool {
	webContainer := GetCurrentRunningWebContainer()
	if webContainer != nil {
		ok := RunContainerShellScript("/itops/etc/web/init.sh") // 执行web初始化脚本
		audit := fmt.Sprintf("发送初始化请求到容器：%v", ok)
		fmt.Println(audit)
		return ok
	}
	return false
}
