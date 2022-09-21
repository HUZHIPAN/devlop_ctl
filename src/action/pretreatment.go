package action

import (
	"fmt"
	"io/fs"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/src/common"
	"os"
	"path/filepath"
	"strings"
)

var (
	EtcBranchRuntimeSuffix = "_active" // etc配置包预处理后（运行时）的分支， 每个都对应一个去掉的后缀的分支是预处理前的分支
)

// 根据一组宏值预处理
// 每次预处理以不带 _active 后缀的分支开始
// 将预处理后的文件变动提交的当前分支加 _active 后缀的分支
func PreProcessMacros(vHostsMacros, webScriptMacros, startMacros ,runcConfMacros map[string]string) error {
	CheckAndCommitEtcChange()
	etcPath := common.GetEtcPath()
	currentBranch := gogit.GetRepositoryCurrentBranch(etcPath)

	if strings.Contains(currentBranch, EtcBranchRuntimeSuffix) { // 已处于处理后分支时，先切换至处理前分支
		diary.Infof("配置目录仓库（%v）当前处于预处理后分支：%v", etcPath, currentBranch)
		if !gogit.CheckoutRepositoryBranch(etcPath, strings.TrimSuffix(currentBranch, EtcBranchRuntimeSuffix), []string{}) {
			return fmt.Errorf("预处理nginx配置文件：切换分支发生异常")
		} else {
			diary.Infof("切换配置目录仓库至当前预处理后分支对应的处理前分支（%v）成功", strings.TrimSuffix(currentBranch, EtcBranchRuntimeSuffix))
		}
	}

	befeorBranch := strings.TrimSuffix(currentBranch, EtcBranchRuntimeSuffix) // 处理前分支
	aftherBranch := befeorBranch + EtcBranchRuntimeSuffix                     // 处理后分支

	if gogit.CheckBranchIsExist(etcPath, aftherBranch) { // 当前配置分支已存在预处理后的分支，先删除
		if !gogit.DeleteBranchName(etcPath, aftherBranch) {
			return fmt.Errorf("预处理nginx配置文件：删除分支（%v）发生异常", aftherBranch)
		} else {
			diary.Infof("删除配置目录仓库预处理后分支（%v）成功", aftherBranch)
		}
	}

	if !gogit.CreateRepositoryFromByCurrentHead(etcPath, aftherBranch) {
		return fmt.Errorf("预处理nginx配置文件：创建分支（%v）发生异常", aftherBranch)
	} else {
		diary.Infof("创建配置目录仓库预处理后分支（%v）成功", aftherBranch)
	}
	if !gogit.CheckoutRepositoryBranch(etcPath, aftherBranch, []string{}) {
		return fmt.Errorf("预处理nginx配置文件：切换至分支（%v）发生异常", aftherBranch)
	} else {
		diary.Infof("配置目录仓库切换至预处理后分支（%v）成功", aftherBranch)
	}

	vHostsPath := common.GetEtcPath() + "/nginx/vhosts/"
	existError := replaceDirectionMacros(vHostsPath, vHostsMacros)
	if existError != nil {
		diary.Errorf("处理nginx配置文件发生错误：%v，宏：%v", existError, vHostsMacros)
		return existError
	} else {
		diary.Infof("处理替换nginx配置文件：%v", vHostsMacros)
	}

	webScriptPath := common.GetEtcPath() + "/web/"
	err := replaceDirectionMacros(webScriptPath, webScriptMacros)
	if err != nil {
		diary.Errorf("处理web脚本目录宏替换发生错误：%v，宏：%v", err, webScriptMacros)
		return err
	} else {
		diary.Infof("处理web脚本目录宏替换：%v", webScriptMacros)
	}

	startScriptFile := common.GetEtcPath() + "/start.sh"
	err = replaceFileMacros(startScriptFile, startMacros)
	if err != nil {
		diary.Errorf("处理容器启动脚本（%v）宏替换发生错误：%v，宏：%v", startScriptFile, err, webScriptMacros)
		return err
	} else {
		diary.Infof("处理容器启动脚本（%v）宏替换成功，宏：%v", startScriptFile, webScriptMacros)
	}

	runcConfigFile := common.GetEtcRuncPath() + "/config.json"
	err = replaceFileMacros(runcConfigFile, runcConfMacros)
	if err != nil {
		diary.Errorf("处理runc启动配置文件（%v）宏替换发生错误：%v，宏：%v", runcConfigFile, err, runcConfMacros)
		return err
	} else {
		diary.Infof("处理runc启动配置文件（%v）宏替换成功，宏：%v", runcConfigFile, runcConfMacros)
	}

	changeList, err := gogit.RepositoryWorkSpaceStatus(etcPath, []string{})
	if err != nil {
		diary.Errorf("获取配置目录仓库预处理后的变动文件列表失败！")
	}
	ok := gogit.CommitDirChange(etcPath, fmt.Sprintf("预处理完成后提交，vhosts宏：%v, web脚本宏：%v", vHostsMacros, webScriptMacros), []string{})
	if !ok {
		return fmt.Errorf("提交预处理后分支改动失败，%v个文件变动", len(changeList))
	} else {
		diary.Infof("提交预处理后分支改动成功，%v个文件变动", len(changeList))
	}

	return nil
}

// 对一个目录的文件进行宏替换
func replaceDirectionMacros(directionPath string, macros map[string]string) error {
	existError := filepath.Walk(directionPath, func(path string, info fs.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		return replaceFileMacros(path, macros)
	})
	return existError
}

// 替换单个文件中的宏
func replaceFileMacros(filename string, macros map[string]string) error {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	content := string(fileContent)
	for macroName, val := range macros {
		content = strings.ReplaceAll(content, macroName, val)
	}

	return writeFileOver(filename, &content)
}

// 覆盖写入文件
func writeFileOver(path string, content *string) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(*content)
	return err
}
