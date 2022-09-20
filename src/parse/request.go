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

	"github.com/docker/docker/api/types"
)

// 更新环境操作
func ApplyCommandHandle(params *structure.ApplyParams) bool {
	// 从已存在的lwjk_app目录加载项目
	if params.LoadWithAppPath != "" && util.FileExists(params.LoadWithAppPath) {
		diary.Infof("开始从已存在的项目lwjk_app目录（%v）加载，加载为版本号：%v", params.LoadWithAppPath, params.LoadAppVersion)
		ok := action.ApplyWithExistDirectory(params.LoadWithAppPath, params.LoadAppVersion)
		if !ok {
			fmt.Printf("从目录（%v）加载产品代码失败！\n", params.LoadWithAppPath)
			return false
		}

		fmt.Printf("从目录（%v）加载产品代码成功，版本号为：%v\n", params.LoadWithAppPath, params.LoadAppVersion)
		return true
	}

	/************解析并更新包操作*************/
	applyHandle := ParseRequestPackage(params.PackagePath)
	if applyHandle == nil {
		os.RemoveAll(getPackageFileUnpackPath(params.PackagePath))
		fmt.Println("更新包校验失败！")
		return false
	}

	diary.Infof("解析yaml配置：%v", applyHandle.GetYamlDesc())
	applyHandle.execute()

	defer fmt.Println("更新操作完成！")

	currentTimeStr := time.Now().Format("2006-01-02_15_04_05")
	auditFileName := strings.TrimSuffix(path.Base(params.PackagePath), ".tar.gz") + "_apply_audit_" + currentTimeStr
	auditFileNamePath := common.GetTmpPath() + "/logs/" + auditFileName + ".txt"
	ok, err := util.WriteFileWithDir(auditFileNamePath, diary.Ob_get_contents())
	if err != nil || !ok {
		defer fmt.Println("更新审计保存失败：", err)
	}

	defer fmt.Println("审计文件：", auditFileNamePath)
	return true
}

// 生成WEB容器和环境配置
func BuildCommandHandle(params *structure.BuildParams) bool {
	existContainer := action.GetCurrentExistWebContainer()
	if existContainer != nil {
		if action.RemoveWebContainer() {
			fmt.Println("当前部署目录存在相关容器，已自动删除")
		} else {
			defer fmt.Println("当前部署目录存在相关容器，自动删除失败！")
			return false
		}
	}

	currentImageNumber := action.GetLastWebImageTagNumber()
	if currentImageNumber <= 0 {
		fmt.Println("当前部署环境缺少image镜像包，请先更新基础镜像包！")
		return false
	}

	etcCurrentBranch := gogit.GetRepositoryCurrentBranch(common.GetEtcPath())
	if etcCurrentBranch == "" || etcCurrentBranch == "master" {
		fmt.Println("当前部署目录缺少configure配置包，请先更新配置包！")
		return false
	}

	if !util.FileExists(common.GetPersistenceVolume()) {
		fmt.Printf("持久化目录（%v）不存在，请确保当前环境已经更新过产品包！\n", common.GetPersistenceVolume())
		return false
	}

	useUid := action.GetUseRunContainerUserUid()
	err := action.PreProcessMacros(map[string]string{
		"{{$WEB_API_PORT}}": fmt.Sprintf("%d", params.WebApiPort),
		"{{$WEB_PORT}}":     fmt.Sprintf("%d", params.WebPort),
	}, map[string]string{
		"{{$BACKEND_API_GATEWAY}}": params.WebApiGateway,
		"{{$PLACEHOLDER}}":         "",
	}, map[string]string{
		"{{$UID}}": fmt.Sprintf("%d", useUid),
	})
	if err != nil {
		fmt.Println("容器启动配置etc配置预处理失败：", err)
		return false
	}

	webContainer := action.CreateContainer(params)
	if webContainer != nil {
		if params.MacAddr == "" {
			params.MacAddr = getContainerMacAddress(webContainer) // 记录生成的mac_addr
		}
		GenerateEnvBuildParams(params, useUid)
		ShowWebStatus()
		fmt.Println("创建容器成功：", webContainer.ID)
		return true
	} else {
		fmt.Println(diary.Ob_get_contents())
		fmt.Println("创建容器失败！")
		return false
	}
}

// WEB容器状态管理（启动、停止、重启、查看），不包含创建WEB容器
func WebCommandHandle(params *structure.WebParams) bool {
	if !util.InArray(params.Action, []string{"start", "stop", "restart", "status", "enter"}) {
		fmt.Printf("未知的操作：%v\n", params.Action)
		return false
	}

	switch params.Action {
	case "status":
		ShowWebStatus()
	case "start":
		ok := action.RunContainer()
		if ok {
			defer fmt.Println("启动成功！")
			ShowWebStatus()
			action.RunAppInitializationCommand()
		} else {
			fmt.Println(diary.Ob_get_contents())
			fmt.Println("启动失败！")
		}

	case "stop":
		stopOk := action.StopContainer()
		if stopOk {
			fmt.Println("关闭容器成功")
		}
		if params.WithRemove {
			action.RemoveWebContainer()
		}

	case "restart":
		_ = action.StopContainer()
		ok := action.RunContainer()
		if ok {
			fmt.Println("重启成功！")
			ShowWebStatus()
			action.RunAppInitializationCommand()
		} else {
			fmt.Println(diary.Ob_get_contents())
		}

	case "enter":
		webContainer := action.GetCurrentExistWebContainer()
		if webContainer == nil {
			fmt.Println("当前部署目录未创建WEB容器！（使用 `lwctl build` 命令创建容器）")
			return false
		}
		exitCode := util.RunCommandWithCli("docker", "exec", "-it", webContainer.ID, "bash")
		fmt.Printf("已退出WEB容器，exitCode：%d \n", exitCode)
	}

	return true
}

// 显示web容器状态
func ShowWebStatus() {
	webContainer := action.GetCurrentExistWebContainer()
	if webContainer != nil {
		envParams := GetDeployEnvParams()
		fmt.Printf("WEB容器名称：%v \n", webContainer.Names)
		fmt.Printf("容器ID：%v \n", webContainer.ID)
		fmt.Printf("部署路径：%v \n", common.GetLwopsVolume())
		if strings.Contains(webContainer.Status, "Up") {
			fmt.Printf("使用端口：%v \n", webContainer.Ports)
			defer fmt.Println("WEB容器正在运行中！")
			if macAddr := getContainerMacAddress(webContainer); macAddr != "" {
				fmt.Printf("物理地址（MAC地址）：%v \n", macAddr)
			}
		} else {
			defer fmt.Println("未启动WEB容器！（使用 `lwctl web -s start` 启动容器）")
		}

		fmt.Printf("使用镜像：%v \n", webContainer.Image)
		fmt.Printf("生成配置：%v \n", envParams.Build)
		fmt.Printf("创建时间：%v \n", time.Unix(webContainer.Created, 0).Format("2006-01-02 15:04:05"))
		fmt.Printf("容器状态：%v \n", webContainer.Status)
	} else {
		defer fmt.Println("部署目录未创建WEB容器！（请使用 `lwctl build` 命令创建容器）")
	}
}

func getContainerMacAddress(webContainer *types.Container) string {
	endpointSetting := webContainer.NetworkSettings.Networks["bridge"]
	if endpointSetting != nil {
		return endpointSetting.MacAddress
	}
	return ""
}

// 回滚操作
func RollbackCommandHandle(params *structure.RollbackParams) bool {
	if !util.InArray(params.Type, []string{"image"}) {
		fmt.Printf("未知的操作：%v\n", params.Type)
		return false
	}

	switch params.Type {
	case "image":
		action.RollbackLastImage()
	}

	return true
}

// 部署更新的产品版本管理（列出 或 切换）
func AppCommandHandle(params *structure.AppParams) bool {
	currentBranchName := gogit.GetRepositoryCurrentBranch(common.GetLwappPath())
	branchList := gogit.GetRepositoryBranchList(common.GetLwappPath())

	if params.ShowVersionList && params.ToVersion == "" {
		fmt.Printf("当前应用版本：%v \n", currentBranchName)
		fmt.Println("已安装版本：")
		for _, branchItem := range branchList {
			if branchItem == "master" {
				continue
			}
			appVersionInfo := common.GetAppVersionNumberInfo(branchItem)
			fmt.Printf("部署方式：%-8s版本号：%v \n", appVersionInfo.Mode, appVersionInfo.VersionNumber)
		}
		return true
	}

	if params.ToVersion == "" {
		fmt.Println("请输入要切换的版本！")
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
		fmt.Println("当前部署目录不存在版本号：", params.ToVersion)
		return false
	}

	isRunningContainer := action.GetCurrentRunningWebContainer() != nil
	if isRunningContainer {
		fmt.Println("当前WEB容器正在运行中！切换版本操作会先停止WEB容器运行，将在版本切换完成后重启")
		action.StopContainer() // 切换版本会先停止容器
	}

	autoCommit := action.CheckAndCommitLwappChange() // 检查未忽略未提交的文件自动提交
	if autoCommit > 0 {
		fmt.Printf("切换版本前自动提交%v个未忽略未提交的文件\n", autoCommit)
	}
	ok := action.CheckoutAppVersion(params.ToVersion)
	if ok {
		fmt.Printf("切换至（%v）成功，更新前版本:（%v）\n", strings.Replace(params.ToVersion, "_", " ", 1), strings.Replace(currentBranchName, "_", " ", 1))
		action.CheckAndCreatePersistenceDir()
		common.ChownDirectoryPower(common.GetLwappPath())
	} else {
		fmt.Println(diary.Ob_get_contents())
		fmt.Printf("切换至（%v）失败！当前版本:（%v）\n", strings.Replace(params.ToVersion, "_", " ", 1), strings.Replace(currentBranchName, "_", " ", 1))
	}

	if isRunningContainer {
		if action.RunContainer() {
			fmt.Println("容器重新启动成功")
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
		fmt.Printf("当前应用配置分支：%v \n", currentBranchName)
		fmt.Println("已存在配置版本：")
		for _, branchItem := range branchList {
			if branchItem == "master" || strings.HasSuffix(branchItem, action.EtcBranchRuntimeSuffix) {
				continue
			}
			fmt.Printf("%v \n", branchItem)
		}
		return true
	}

	if params.ToVersion == "" {
		fmt.Println("请输入要切换的配置版本！")
		return false
	}
	if !gogit.CheckBranchIsExist(etcPath, params.ToVersion) {
		fmt.Printf("切换失败，配置版本（%v）不存在！\n", params.ToVersion)
		return false
	}

	if changeNum := action.CheckAndCommitEtcChange(); changeNum > 0 {
		fmt.Printf("切换前自动提交配置包变动%v个文件！\n", changeNum)
	}
	existWebContainer := action.GetCurrentExistWebContainer() != nil          // 更新前存在创建的WEB容器
	existRunningWebContainer := action.GetCurrentRunningWebContainer() != nil // 更新前存在运行中的WEB容器
	if existRunningWebContainer {
		fmt.Println("当前WEB容器运行中，切换配置版本会重新生成WEB容器并重启！")
	}

	if gogit.CheckoutRepositoryBranch(etcPath, params.ToVersion, []string{}) {
		fmt.Printf("切换至配置包版本（%v）成功，切换前版本（%v）\n", params.ToVersion, currentBranchName)
	} else {
		fmt.Printf("切换至配置包版本（%v）失败，当前版本（%v）\n", params.ToVersion, currentBranchName)
		return false
	}

	common.ChownDirectoryPower(common.GetEtcPath())
	// 存在生成容器环境配置 并且 存在创建的WEB容器
	if util.FileExists(getDeployEnvFilePath()) && existWebContainer {
		if existRunningWebContainer {
			action.StopContainer()
		}
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
		fmt.Println("WEB容器未启动！")
		return false
	}

	diary.IsRealTimeOutput = true
	ok := action.RunContainerCommand(params.Command, 0)
	fmt.Println("发送结果：", ok)
	return ok
}
