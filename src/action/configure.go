package action

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/pkg/packer"
	"lwapp/src/common"
)

type ConfigureApplyResult struct {
	IsSuccess bool
	ExistDiff bool
}

func ConfigureUpdateApply(sourcePath string, event *EventPackage) *ConfigureApplyResult {
	result := &ConfigureApplyResult{
		IsSuccess: false,
	}

	etcPath := common.GetEtcPath()
	etcVersion := event.Name
	currentBranchName := gogit.GetRepositoryCurrentBranch(etcPath)
	if currentBranchName != event.Name {
		if !gogit.CheckBranchIsExist(etcPath, etcVersion) {
			if !gogit.CreateRepositoryBranchFromByMaster(etcPath, etcVersion) {
				diary.Errorf("仓库（%v）无法为配置包创建版本分支（%v）！", etcPath, etcVersion)
				return result
			} else {
				diary.Infof("仓库（%v）创建配置包版本分支（%v）成功", etcPath, etcVersion)
			}
		} else {
			diary.Warningf("仓库（%v）配置包版本分支（%v）已存在！", etcPath, etcVersion)
		}

		if !gogit.CheckoutRepositoryBranch(etcPath, etcVersion, []string{}) {
			diary.Errorf("仓库（%v）切换至配置包版本分支（%v）失败！", etcPath, etcVersion)
			return result
		} else {
			diary.Infof("仓库（%v）切换至配置包版本分支（%v）成功", etcPath, etcVersion)
		}
	}

	defer gogit.CleanRepositoryWorkspaceChange(etcPath, []string{}) // 回滚未提交变动
	err := packer.NewTgzPacker().UnPack(sourcePath+"/"+event.FileRelativePath, etcPath, []string{".git/"}, []string{})
	if err != nil {
		diary.Errorf("更新配置包，解压至（%v）目录时发生异常: %v", etcPath, err)
		return result
	} else {
		diary.Errorf("更新配置包，解压至（%v）目录成功", etcPath)
	}

	changeList, err := gogit.RepositoryWorkSpaceStatus(etcPath, []string{})
	if err != nil {
		diary.Errorf("仓库（%v）获取工作区变动发生异常: %v", etcPath, err)
		gogit.CleanRepositoryWorkspaceChange(etcPath, []string{})
		return result
	}

	commitComment := fmt.Sprintf("配置包更新：%v,更新版本：%v,更新描述：%v", event.Type, etcVersion, event.Description)
	if !gogit.CommitDirChange(etcPath, commitComment, []string{}) {
		diary.Errorf("仓库（%v）提交配置包更新变动时失败！%v个文件变动", etcPath, len(changeList))
		return result
	} else {
		diary.Errorf("仓库（%v）提交配置包更新变动成功，%v个文件变动", etcPath, len(changeList))
	}
	result.IsSuccess = true
	diary.Infof("更新配置更新包成功")
	return result
}

func CheckAndCommitEtcChange() int {
	etcPath := common.GetEtcPath()
	workTreeStatus, err := gogit.RepositoryWorkSpaceStatus(etcPath, []string{})
	if err != nil {
		return 0
	}
	if len(workTreeStatus) > 0 {
		if !gogit.CommitDirChange(etcPath, "来源于配置更新包之外的变动，更新前自动提交", []string{}) {
			diary.Errorf("配置更新前自动提交变动失败：\n%v", workTreeStatus)
			return len(workTreeStatus)
		} else {
			diary.Infof("来源于配置更新包之外的变动，更新前自动提交成功：\n%v", workTreeStatus)
		}
	}

	return 0
}

func RollbackLastConfigureUpdate() bool {
	etcPath := common.GetEtcPath()
	ok := gogit.RollbackDirChangeSpan(etcPath, 1)
	if !ok {
		diary.Errorf("回滚最后一次配置更新包失败：%v", etcPath)
		return false
	} else {
		diary.Infof("回滚最后一次配置更新包成功")
	}
	return true
}
