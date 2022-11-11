package parse

import (
	"lwapp/pkg/diary"
	"lwapp/src/action"
	"lwapp/src/common"
	"os"
	"sync"
)

type ApplyHandle struct {
	RequestParams action.RequestParams
	SourcePath    string
	Action        struct {
		Image,
		Configure,
		Product,
		Feature,
		Customer *action.EventPackage
	}
}

func (h *ApplyHandle) GetYamlDesc() [][]string {
	items := [][]string{}

	var upgrade *action.EventPackage

	if upgrade = h.Action.Image; upgrade != nil {
		items = append(items, []string{upgrade.Type, upgrade.Name, upgrade.Description})
	}
	if upgrade = h.Action.Configure; upgrade != nil {
		items = append(items, []string{upgrade.Type, upgrade.Name, upgrade.Description})
	}
	if upgrade = h.Action.Product; upgrade != nil {
		items = append(items, []string{upgrade.Type, upgrade.Name, upgrade.Description})
	}
	if upgrade = h.Action.Feature; upgrade != nil {
		items = append(items, []string{upgrade.Type, upgrade.Name, upgrade.Description})
	}

	commands := []string{"commands"} // 显示解析的命令
	commands = append(commands, h.RequestParams.Metadata.Commands...)
	items = append(items, commands)

	return items
}

func (h *ApplyHandle) SetSourcePath(sourcePath string) {
	h.SourcePath = sourcePath
}
func (h *ApplyHandle) SetRequestParams(requestParams action.RequestParams) {
	h.RequestParams = requestParams
}

func (h *ApplyHandle) LoadByEventPackages(eventPackages []action.EventPackage) {
	for k, eventPackage := range eventPackages {
		switch eventPackage.Type {
		case "product":
			h.Action.Product = &eventPackages[k]
		case "feature":
			h.Action.Feature = &eventPackages[k]
		case "customer":
			h.Action.Customer = &eventPackages[k]
		case "configure":
			h.Action.Configure = &eventPackages[k]
		case "image":
			h.Action.Image = &eventPackages[k]
		}
	}
}

func (h *ApplyHandle) Execute() bool {
	action.CheckAndCommitLwappChange()
	action.CheckAndCommitEtcChange()

	existRunningWebContainer := action.GetCurrentRunningWebContainer() != nil // 更新前存在运行中的WEB容器
	if existRunningWebContainer && h.IsNeedToRestart() {
		diary.Infof("当前部署目录（%v）存在运行中的容器，当前更新操作需要重启容器！", common.GetLwopsVolume())
		action.StopContainer()
	}

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		if h.Action.Image != nil {
			action.ImageUpdateApply(h.SourcePath, h.Action.Image)
		}
		wg.Done()
	}()
	go func() {
		if h.Action.Configure != nil {
			action.ConfigureUpdateApply(h.SourcePath, h.Action.Configure)
		}
		wg.Done()
	}()
	// 全量更新包与增量包需要同步按顺序更新
	go func() {
		if h.Action.Product != nil {
			r := action.ProductUpdateApply(h.SourcePath, h.Action.Product)
			if !r.IsSuccess {
				wg.Done()
				return
			}
		}
		if h.Action.Feature != nil {
			action.FeatureUpdateApply(h.SourcePath, h.Action.Feature)
		}
		wg.Done()
	}()

	wg.Wait()

	wg.Add(1)
	go func() {
		os.RemoveAll(h.SourcePath)
		wg.Done()
	}()

	common.ChownDirectoryPower(common.GetDeploymentLogPath())

	// 存在生成容器环境配置 并且 存在创建的WEB容器
	if common.IsExistDeployEnvSetting() {
		// 当前操作存在 镜像更新 或 配置包更新时 需要重新生成启动容器
		if h.Action.Image != nil || h.Action.Configure != nil {
			diary.Infof("当前更新操作包含镜像或配置更新，尝试重新生成运行配置")
			BuildContainerByEnvParams()  // 根据当前环境配置重新生成容器
		}
	}

	if existRunningWebContainer && h.IsNeedToRestart() {
		if action.RunContainer() { // 启动容器
			action.RunAppInitializationCommand() // 发送初始化命令
		}
	}

	// 更新附带的命令执行
	commands := h.RequestParams.Metadata.Commands
	if len(commands) > 0 && existRunningWebContainer {
		for i, command := range commands {
			action.RunContainerCommand(command, false,(i+1)*5) // 每条命令之间间隔5s执行，防止并行
		}
	}

	if len(commands) > 0 && !existRunningWebContainer {
		diary.Infof("当前环境不存在运行中的容器，已忽略commands执行：%v", commands)
	}

	wg.Wait()
	return true
}

// 此次更新是否需要重启
func (h *ApplyHandle) IsNeedToRestart() bool {
	return h.Action.Configure != nil || h.Action.Image != nil || h.Action.Product != nil
}