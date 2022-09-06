package action

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/pkg/packer"
	"lwapp/src/common"
	"strings"
)

type FeatureApplyResult struct {
	IsSuccess bool
	ExistDiff bool
}

func FeatureUpdateApply(sourcePath string, event *EventPackage) *FeatureApplyResult {
	result := &FeatureApplyResult{
		IsSuccess: false,
	}

	lwAppPath := common.GetLwappPath()
	currentBranchName := gogit.GetRepositoryCurrentBranch(lwAppPath)
	if currentBranchName == "" {
		diary.Errorf("仓库（%v）无法获取当前分支！", lwAppPath)
		return result
	}
	if currentBranchName == "master" {
		diary.Errorf("仓库（%v）不存在部署产品全量包，无法更新增量包！", lwAppPath)
		return result
	}

	// 更新版本（增量）
	incrementVersion := common.IncrPackagePrefix + event.Name
	if GetAppVersionByBranchName(currentBranchName) != event.Name {
		if !gogit.CheckBranchIsExist(lwAppPath, incrementVersion) {
			if !gogit.CreateRepositoryFromByCurrentHead(lwAppPath, incrementVersion) {
				diary.Errorf("仓库（%v）无法为增量新版本创建分支：%v", lwAppPath, incrementVersion)
				return result
			} else {
				diary.Infof("仓库（%v）基于（%v）创建增量版本分支（%v）成功", lwAppPath, currentBranchName, incrementVersion)
			}
		} else {
			diary.Warningf("仓库（%v）增量版本分支（%v）已经存在！", lwAppPath, incrementVersion)
		}

		if !gogit.CheckoutRepositoryBranch(lwAppPath, incrementVersion, common.GetLwappIgnoreExpression()) {
			diary.Errorf("仓库（%v）切换至增量版本分支（%v）失败！", lwAppPath, incrementVersion)
			return result
		} else {
			diary.Infof("仓库（%v）切换至增量版本分支（%v）成功", lwAppPath, incrementVersion)
		}
	} else {
		// 特殊处理不创建新分支
		diary.Infof("仓库（%v）增量包版本与当前产品包版本号相同，在产品分支上进行更新操作！", lwAppPath)
	}

	err := packer.NewTgzPacker().UnPack(sourcePath+"/"+event.FileRelativePath, lwAppPath, GetFeaturePackageIgnoreExpression(), []string{".gitignore"})
	if err != nil {
		diary.Errorf("更新增量代码包操作，解压至（%v）目录时发生异常: %v", lwAppPath, err)
		RollbackProductOnFail(currentBranchName, false)
		return result
	} else {
		diary.Infof("更新增量代码包操作，解压至（%v）目录成功", lwAppPath)
	}
	statusList, err := gogit.RepositoryWorkSpaceStatus(lwAppPath, common.GetLwappIgnoreExpression())
	if err != nil {
		diary.Errorf("仓库（%v）获取工作区变动发生异常: %v", lwAppPath, err)
		RollbackProductOnFail(currentBranchName, false)
		return result
	}
	if len(statusList) > 0 {
		baseBranch := currentBranchName
		if baseBranch == "HEAD" {
			baseBranch = gogit.GetRepositoryCurrentHeadRefHash(lwAppPath) // 基于分支的某一次提交
			diary.Warningf("仓库（%v）更新增量更新包时未处于版本分支，处于提交commit：%v", lwAppPath, baseBranch)
		}

		commitComment := fmt.Sprintf("基于（%v）更新增量包，更新版本：%v,更新描述：%v", baseBranch, event.Name, event.Description)
		if !gogit.CommitDirChange(lwAppPath, commitComment, common.GetLwappIgnoreExpression()) {
			diary.Errorf("仓库（%v）提交增量更新代码时发生异常！ %v个文件变动", lwAppPath, len(statusList))
			RollbackProductOnFail(currentBranchName, false)
			return result
		} else {
			diary.Errorf("仓库（%v）提交增量更新代码成功, %v个文件变动", lwAppPath, len(statusList))
		}
	} else {
		diary.Infof("仓库（%v）产品增量版本更新包没有未忽略的文件变动，已略过提交", lwAppPath)
	}

	CheckAndCreatePersistenceDir()
	result.ExistDiff = true
	result.IsSuccess = true
	return result
}

func GetFeaturePackageIgnoreExpression() []string {
	ignores := []string{}
	for _, item := range common.GetLwappIgnoreExpression() {
		if strings.Contains(item, "web/z") {
			continue
		}
		ignores = append(ignores, item)
	}

	ignores = append(ignores, ".git/")
	return ignores
}
