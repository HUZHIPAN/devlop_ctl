package parse

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/pkg/util"
	"lwapp/src/action"
	"lwapp/src/common"
	"lwapp/src/structure"
	"os"
	"path"
	"strings"
	"time"
)

// 更新环境操作
func ApplyCommandHandle(params *structure.ApplyParams) bool {
	// 从已存在的lwjk_app目录加载项目
	if params.LoadWithAppPath != "" && util.FileExists(params.LoadWithAppPath) {
		diary.Infof("开始从已存在的项目lwjk_app目录（%v）加载，加载为版本号：%v", params.LoadWithAppPath, params.LoadAppVersion)
		ok := action.ApplyWithExistDirectory(params.LoadWithAppPath, params.LoadAppVersion)
		if !ok {
			diary.Infof("从目录（%v）加载产品代码失败！", params.LoadWithAppPath)
			return false
		}

		diary.Infof("从目录（%v）加载产品代码成功，版本号为：%v", params.LoadWithAppPath, params.LoadAppVersion)
		return true
	}

	/************解析并更新包操作*************/
	applyHandle := ParseRequestPackage(params.PackagePath)
	if applyHandle == nil {
		os.RemoveAll(GetPackageFileUnpackPath(params.PackagePath))
		diary.Errorf("更新包校验失败！")
		return false
	}

	diary.Infof("解析yaml配置：%v", applyHandle.GetYamlDesc())
	applyHandle.Execute()

	auditFileName := strings.TrimSuffix(path.Base(params.PackagePath), ".tar.gz") + "_apply_audit_" + time.Now().Format("2006-01-02_15_04_05")
	auditFileNamePath := common.GetTmpPath() + "/logs/" + auditFileName + ".txt"
	ok, err := util.WriteFileWithDir(auditFileNamePath, diary.Ob_get_contents())
	if err != nil || !ok {
		diary.Infof("更新审计保存失败：%v", err)
	} else {
		diary.Infof("审计文件：%v", auditFileNamePath)
	}


	diary.Infof("更新操作完成！")
	return true
}

// 生成WEB容器和环境配置
func BuildCommandHandle(params *structure.BuildParams) bool {
	webContainer := action.GetCurrentRunningWebContainer()
	if webContainer != nil {
		diary.Warningf("当前部署目录相关容器正在运行中！，请先手动停止容器运行！")
		return false
	}

	currentImageNumber := action.GetLastWebImageTagNumber()
	if currentImageNumber <= 0 {
		diary.Warningf("当前部署环境缺少image镜像包，请先更新基础镜像包！")
		return false
	}

	etcCurrentBranch := gogit.GetRepositoryCurrentBranch(common.GetEtcPath())
	if etcCurrentBranch == "" || etcCurrentBranch == "master" {
		diary.Warningf("当前部署目录缺少configure配置包，请先更新配置包！")
		return false
	}

	if !util.FileExists(common.GetPersistenceVolume()) {
		diary.Warningf("持久化目录（%v）不存在，请确保当前环境已经更新过产品包！", common.GetPersistenceVolume())
		return false
	}

	useUid := action.GetUseRunContainerUserUid()
	err := action.PreProcessMacros(map[string]string{
		"{{$WEB_API_PORT}}":         fmt.Sprintf("%d", params.WebApiPort),
		"{{$WEB_PORT}}":             fmt.Sprintf("%d", params.WebPort),
		"{{$PHPFPM_LISTEN_SOCKET}}": common.GetPhpfpmSocketListenFile(),
	}, map[string]string{
		"{{$BACKEND_API_GATEWAY}}": params.WebApiGateway,
		"{{$PLACEHOLDER}}":         "",
	}, map[string]string{
		"{{$UID}}": fmt.Sprintf("%d", useUid),
	}, map[string]string{
		"{{$IMAGE_ROOTFS_PATH}}":     fmt.Sprintf("%v/%v", common.GetRootfsPath(), action.GetLastWebImageTagNumber()),
		"{{$UID}}":                   fmt.Sprintf("%d", useUid),
		"{{$LWJK_APP_PATH}}":         common.GetLwappPath(),
		"{{$WEB_ETC_PATH}}":          common.GetEtcPath(),
		"{{$WEB_LOGS_PATH}}":         common.GetDeploymentLogPath(),
		"{{$PersistenceVolumePATH}}": common.GetPersistenceVolume(),
	}, map[string]string{
		"{{$PHPFPM_LISTEN_SOCKET}}": common.GetPhpfpmSocketListenFile(),
	})

	if err != nil {
		diary.Errorf("容器启动配置etc配置预处理失败：", err)
		return false
	}

	GenerateEnvBuildParams(params, useUid)
	diary.Infof("生成容器运行环境配置：%+v ", *params)
	diary.Infof("\n生成环境配置成功！")
	return true
}

// WEB容器状态管理（启动、停止、重启、查看），不包含创建WEB容器
func WebCommandHandle(params *structure.WebParams) bool {
	if !util.InArray(params.Action, []string{"start", "stop", "restart", "status", "enter"}) {
		diary.Warningf("未知的操作：%v\n", params.Action)
		return false
	}

	switch params.Action {
	case "status":
		running := ShowWebStatus()
		if running {
			diary.Infof("\n容器正在运行中！")
		} else {
			diary.Infof("\n当前部署目录容器未启动！")
		}
	case "start":
		ok := action.RunContainer()
		if ok {
			ShowWebStatus()
			action.RunAppInitializationCommand()
			diary.Infof("\n启动成功！")
		} else {
			diary.Errorf("\n启动失败！")
		}
	case "stop":
		stopOk := action.StopContainer()
		diary.Infof("")
		if stopOk {
			diary.Infof("关闭容器成功！")
		}
	case "restart":
		_ = action.StopContainer()
		ok := action.RunContainer()
		if ok {
			ShowWebStatus()
			action.RunAppInitializationCommand()
			diary.Infof("\n重启成功！")
		} else {
			diary.Errorf("\n重启发生错误！")
		}

	case "enter":
		webContainer := action.GetCurrentRunningWebContainer()
		if webContainer == nil {
			diary.Warningf("当前部署目录WEB容器未启动！")
			return false
		}
		common.UnlockLwopsEnv()
		diary.Infof("将进入WEB容器内操作，默认登录itops用户，可在容器内使用`su root`切换至root用户！\n\n\n")
		execUser := fmt.Sprintf("%d:%d", action.GetUseRunContainerUserUid(), action.GetUseRunContainerUserUid())
		exitCode := util.RunCommandWithCli(action.GetRuncBin(), "--root", action.GetRuncStatePath(), "exec", "--tty", "--user", execUser, webContainer.Name, "bash")
		diary.Infof("已退出WEB容器，exitCode：%d \n\n\n", exitCode)
	}

	return true
}

// 显示web容器状态
func ShowWebStatus() bool {
	envParams := common.GetDeployEnvParams()
	webContainer := action.GetCurrentRunningWebContainer()
	if webContainer != nil {
		diary.Infof("WEB容器运行名称：%v", webContainer.Name)
		diary.Infof("runc进程pid：%v", webContainer.Pid)
		diary.Infof("部署路径：%v", common.GetLwopsVolume())
		diary.Infof("启动命令：%v", util.GetProcessCmdlineParams(webContainer.Pid))
		diary.Infof("环境配置：%+v", envParams.Build)
		return true
	} else {
		if common.IsExistDeployEnvSetting() {
			diary.Infof("环境配置：%+v ", envParams.Build)
		} else {
			diary.Infof("未生成运行环境配置，使用 lwctl build 命令创建运行环境配置！")
		}
		diary.Infof("当前部署目录未启动WEB容器！（使用 `lwctl web -s start` 启动容器）")
		return false
	}
}

// 回滚操作
func RollbackCommandHandle(params *structure.RollbackParams) bool {
	if !util.InArray(params.Type, []string{"image"}) {
		diary.Infof("未知的操作：%v", params.Type)
		return false
	}

	// switch params.Type {
	// case "image":
	// 	RollbackLastImage()
	// }

	return true
}

// 部署更新的产品版本管理（列出 或 切换）
func AppCommandHandle(params *structure.AppParams) bool {
	currentBranchName := gogit.GetRepositoryCurrentBranch(common.GetLwappPath())
	branchList := gogit.GetRepositoryBranchList(common.GetLwappPath())

	if params.ShowVersionList && params.ToVersion == "" {
		diary.Infof("当前应用版本：%v", currentBranchName)
		diary.Infof("已安装版本：")
		for _, branchItem := range branchList {
			if branchItem == "master" {
				continue
			}
			appVersionInfo := common.GetAppVersionNumberInfo(branchItem)
			diary.Infof("部署方式：%-8s版本号：%v", appVersionInfo.Mode, appVersionInfo.VersionNumber)
		}
		return true
	}

	if params.ToVersion == "" {
		diary.Warningf("请输入要切换的版本！")
		return false
	}

	if !util.InArray(params.ToVersion, branchList) {
		if util.InArray(common.IncrPackagePrefix+params.ToVersion, branchList) {
			params.ToVersion = common.IncrPackagePrefix + params.ToVersion // 增量包更新版本号
		}
		if util.InArray(common.FullPackagePrefix+params.ToVersion, branchList) {
			params.ToVersion = common.FullPackagePrefix + params.ToVersion // 全量包更新版本号
		}
	}

	if !gogit.CheckBranchIsExist(common.GetLwappPath(), params.ToVersion) {
		diary.Errorf("当前部署目录不存在版本号：", params.ToVersion)
		return false
	}

	isRunningContainer := action.GetCurrentRunningWebContainer() != nil
	if isRunningContainer {
		diary.Infof("当前WEB容器正在运行中！切换版本操作会先停止WEB容器运行，将在版本切换完成后重启")
		action.StopContainer() // 切换版本会先停止容器
	}

	autoCommit := action.CheckAndCommitLwappChange() // 检查未忽略未提交的文件自动提交
	if autoCommit > 0 {
		diary.Infof("切换版本前自动提交%v个未忽略未提交的文件", autoCommit)
	}
	ok := action.CheckoutAppVersion(params.ToVersion)
	if ok {
		diary.Infof("切换至（%v）成功，更新前版本:（%v）", strings.Replace(params.ToVersion, "_", " ", 1), strings.Replace(currentBranchName, "_", " ", 1))
		action.CheckAndCreatePersistenceDir()
	} else {
		diary.Errorf("切换至（%v）失败！当前版本:（%v）", strings.Replace(params.ToVersion, "_", " ", 1), strings.Replace(currentBranchName, "_", " ", 1))
	}

	if isRunningContainer {
		if action.RunContainer() {
			diary.Infof("容器重新启动成功")
		}
		action.RunAppInitializationCommand()
	}

	return true
}

// 部署更新的配置包版本管理（列出 或 切换）
func EtcCommandHandle(params *structure.EtcParams) bool {
	etcPath := common.GetEtcPath()
	currentBranchName := gogit.GetRepositoryCurrentBranch(etcPath)
	branchList := gogit.GetRepositoryBranchList(etcPath)

	if params.ShowVersionList && params.ToVersion == "" {
		diary.Infof("当前应用配置分支：%v", currentBranchName)
		diary.Infof("已存在配置版本：")
		for _, branchItem := range branchList {
			if branchItem == "master" || strings.HasSuffix(branchItem, action.EtcBranchRuntimeSuffix) {
				continue
			}
			diary.Infof("%v", branchItem)
		}
		return true
	}

	if params.ToVersion == "" {
		diary.Warningf("请输入要切换的配置版本！")
		return false
	}
	if !gogit.CheckBranchIsExist(etcPath, params.ToVersion) {
		diary.Errorf("切换失败，配置版本（%v）不存在！", params.ToVersion)
		return false
	}

	if changeNum := action.CheckAndCommitEtcChange(); changeNum > 0 {
		diary.Warningf("切换前自动提交配置包变动%v个文件！", changeNum)
	}

	existRunningWebContainer := action.GetCurrentRunningWebContainer() != nil // 更新前存在运行中的WEB容器
	if existRunningWebContainer {
		diary.Infof("当前WEB容器运行中，切换配置版本将会停止WEB容器运行！")
		action.StopContainer()
	}

	if gogit.CheckoutRepositoryBranch(etcPath, params.ToVersion, []string{}) {
		diary.Infof("切换至配置包版本（%v）成功，切换前版本（%v）", params.ToVersion, currentBranchName)
	} else {
		diary.Errorf("切换至配置包版本（%v）失败，当前版本（%v）", params.ToVersion, currentBranchName)
		return false
	}

	common.ChownDirectoryPower(common.GetEtcPath())

	if params.WithBuild {
		diary.Infof("切换配置包版本成功，尝试重新生成运行配置并启动容器")
	} else {
		return true
	}


	// 存在生成容器环境配置 并且 存在创建的WEB容器
	if util.FileExists(common.GetDeployEnvFilePath()) {
		if BuildContainerByEnvParams() { // 根据当前环境配置重新生成容器
			if existRunningWebContainer {
				if action.RunContainer() { // 启动容器
					action.RunAppInitializationCommand() // 发送初始化命令
				}
			}
		}
	}

	return true
}

// 执行容器内的命令
func ExecCommandHandle(params *structure.ExecParams) bool {
	webContainer := action.GetCurrentRunningWebContainer()
	if webContainer == nil {
		diary.Errorf("WEB容器未启动！")
		return false
	}
	isBackstage := (util.InArray("-d", os.Args) || util.InArray("--d", os.Args))
	diary.IsRealTimeOutput = true
	ok := action.RunContainerCommand(params.Command, !isBackstage, 0)
	if isBackstage {
		diary.Infof("发送结果：", ok)
	}
	return ok
}
