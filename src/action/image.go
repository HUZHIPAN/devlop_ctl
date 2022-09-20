package action

import (
	"context"
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/docker"
	"lwapp/pkg/gogit"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"os"
	"strconv"
	"strings"
	"time"

	"lwapp/src/structure"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

type ImageApplyResult struct {
	IsSuccess bool
	ExistDiff bool
}

var imageNamePrefix = "lwapp_image_web"

var WebRunContainerName = "lwops_web"

func ImageUpdateApply(sourcePath string, event *EventPackage) *ImageApplyResult {
	result := &ImageApplyResult{
		IsSuccess: false,
	}

	currentVersionNumber := GetLastWebImageTagNumber()
	if currentVersionNumber == -1 {
		return result
	}

	dock := docker.NewDockerClient()
	err := dock.LoadImage(sourcePath + "/" + event.FileRelativePath)
	if err != nil {
		diary.Errorf("加载镜像时发生错误：%v", err)
		return result
	}

	imgTag := event.Name // 更新包镜像tag

	currentTag := fmt.Sprintf("%v:%v", imageNamePrefix, currentVersionNumber) // 当前环境已存在的最新镜像tag

	refreshAfterNumber := currentVersionNumber + 1
	refreshAfterTag := fmt.Sprintf("%v:%v", imageNamePrefix, refreshAfterNumber)

	imgID := dock.GetImageIdByTag(imgTag)
	currentLatestImgID := dock.GetImageIdByTag(currentTag)
	if imgID != "" && (imgID == currentLatestImgID) {
		diary.Infof("加载镜像（%v）镜像ID：%v，与当前最新镜像tag（%v）ID一致，已略过", imgTag, imgID, currentTag)
		result.ExistDiff = false
	} else {
		err = dock.ReTagImage(imgTag, refreshAfterTag)
		if err != nil {
			diary.Errorf("找不到对应的镜像（%v）或镜像tag错误：%v", imgTag, err)
			return result
		} else {
			diary.Infof("加载镜像（%v）成功，重新打标签为（%v）", imgTag, refreshAfterTag)
			result.ExistDiff = true
		}
	}

	result.IsSuccess = true

	diary.Infof("更新镜像包成功")
	return result
}

func RollbackLastImage() bool {
	lastImageNumber := GetLastWebImageTagNumber()

	dock := docker.NewDockerClient()
	LastImageTag := fmt.Sprintf("%v:%v", imageNamePrefix, lastImageNumber)
	err := dock.RemoveImage(LastImageTag)
	if err != nil {
		diary.Errorf("回滚最后一次加载的镜像失败：%v", err)
		return false
	} else {
		diary.Infof("回滚镜像成功：%v", LastImageTag)
	}
	return true
}

func GetLastWebImageTagNumber() int {
	dock := docker.NewDockerClient()
	opt := filters.NewArgs(filters.KeyValuePair{Key: "reference", Value: imageNamePrefix})
	imageList, err := dock.ImageList(context.TODO(), types.ImageListOptions{Filters: opt})
	if err != nil {
		diary.Errorf("获取镜像列表发生错误：%v", err)
		return -1
	}

	var currentMaxVersionNumber int64 = 0
	for _, img := range imageList {
		for _, repoTag := range img.RepoTags {
			number, err := strconv.ParseInt(strings.TrimPrefix(repoTag, imageNamePrefix+":"), 0, 64)
			if err != nil {
				continue
			}
			if number > currentMaxVersionNumber {
				currentMaxVersionNumber = number
			}
		}
	}
	if currentMaxVersionNumber == 0 {
		diary.Errorf("未加载过基础运行镜像！")
	}
	return int(currentMaxVersionNumber)
}

// 执行web容器内的命令，工作目录（/itops/nginx/html/lwjk_app）
func RunContainerCommand(runCmd string, delay int) bool {
	d := docker.NewDockerClient()
	webContainer := GetCurrentRunningWebContainer()
	if webContainer == nil {
		fmt.Printf("发送容器exec命令（%v）错误：没有运行中的WEB容器! \n", runCmd)
		return false
	}

	execLogFileSuffix := time.Now().Format("2006-01-02") + ".txt"
	execLogFile := "/itops/logs/exec/" + execLogFileSuffix
	execLogMessage := fmt.Sprintf("\n%v 执行命令：>>>>>>>>>>>>>>>>>>>>>> %v", time.Now().Format("2006-01-02 15:04:05"), runCmd)

	commandConfig := types.ExecConfig{
		User:         fmt.Sprintf("%d", GetUseRunContainerUserUid()),
		AttachStdin:  false,
		AttachStderr: false,
		AttachStdout: false,
		WorkingDir:   "/itops/nginx/html/lwjk_app",
		Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("sleep %d; echo '%v' >> %v; bash -c '%v' &>> %v", delay, execLogMessage, execLogFile, runCmd, execLogFile)},
	}
	idResponse, err := d.ContainerExecCreate(context.TODO(), webContainer.ID, commandConfig)
	if err != nil {
		fmt.Printf("创建容器exec命令（%v）错误：%v \n", runCmd, err)
		return false
	}

	startCheck := types.ExecStartCheck{}
	err = d.ContainerExecStart(context.TODO(), idResponse.ID, startCheck)
	// response, err := d.ContainerExecAttach(context.TODO(), idResponse.ID, startCheck)
	if err != nil {
		fmt.Printf("发送exec命令（%v）失败：%v", runCmd, err)
		return false
	}

	diary.Infof("发送exec命令（%v）成功，可在（%v）中查看执行结果", runCmd, common.GetWebExecLogPath()+"/"+execLogFileSuffix)
	return true
}

// 执行web容器内的脚本文件，工作目录（/itops/nginx/html/lwjk_app）
func RunContainerShellScript(shFile string) bool {
	d := docker.NewDockerClient()
	webContainer := GetCurrentRunningWebContainer()
	if webContainer == nil {
		return false
	}

	execLogFile := "/itops/logs/exec/" + time.Now().Format("2006-01-02") + "_bash" + ".txt"
	execLogMessage := fmt.Sprintf("\n%v 执行脚本（%v）：>>>>>>>>>>>>>>>>>>>>>>>>>>>>", time.Now().Format("2006-01-02 15:04:05"), shFile)

	commandConfig := types.ExecConfig{
		User:         fmt.Sprintf("%d", GetUseRunContainerUserUid()),
		AttachStdin:  false,
		AttachStderr: false,
		AttachStdout: false,
		WorkingDir:   "/itops/nginx/html/lwjk_app",
		Cmd:          []string{"/bin/sh", "-c", fmt.Sprintf("echo '%v' >> %v; sh %v &>> %v", execLogMessage, execLogFile, shFile, execLogFile)},
	}
	idResponse, err := d.ContainerExecCreate(context.TODO(), webContainer.ID, commandConfig)
	if err != nil {
		fmt.Printf("创建容器exec脚本（%v）发生错误：%v \n", commandConfig.Cmd, err)
		return false
	}

	startCheck := types.ExecStartCheck{}
	err = d.ContainerExecStart(context.TODO(), idResponse.ID, startCheck)
	// response, err := d.ContainerExecAttach(context.TODO(), idResponse.ID, startCheck)
	if err != nil {
		fmt.Printf("发送容器exec脚本（%v）失败：%v \n", commandConfig.Cmd, err)
		return false
	}

	return true
}

// 创建web容器
func CreateContainer(params *structure.BuildParams) *types.Container {
	lastImageTagNumber := GetLastWebImageTagNumber()
	if lastImageTagNumber <= 0 {
		diary.Errorf("创建容器失败：未找到镜像")
		return nil
	}

	webPortStr := fmt.Sprintf("%d", params.WebPort)
	webApiPortStr := fmt.Sprintf("%d", params.WebApiPort)

	d := docker.NewDockerClient()
	c := &container.Config{
		// User:       util.GetCurrentRunUser(),
		WorkingDir: "/itops/nginx/html/lwjk_app",
		MacAddress: params.MacAddr,
		ExposedPorts: nat.PortSet{
			nat.Port(webPortStr + "/tcp"):    {},
			nat.Port(webApiPortStr + "/tcp"): {},
		},
		Cmd:   strslice.StrSlice{"sh", "/itops/etc/start.sh"},
		Image: imageNamePrefix + ":" + fmt.Sprintf("%v", lastImageTagNumber),
	}

	pvPath := common.GetPersistenceVolume()
	if strings.Contains(pvPath, "\\") { // 兼容windows路径，此目录映射在windows下不可用，仅兼容
		pvPath = strings.ReplaceAll(pvPath, "\\", "/")
		pvPath = strings.ReplaceAll(pvPath, ":", "")
		if !strings.HasPrefix(pvPath, "/") {
			pvPath = "/" + pvPath
		}
	}

	h := &container.HostConfig{
		// NetworkMode:   "bridge",
		RestartPolicy: container.RestartPolicy{},
		AutoRemove:    false,
		PortBindings: nat.PortMap{
			nat.Port(webPortStr + "/tcp"): []nat.PortBinding{
				{HostPort: webPortStr},
			},
			nat.Port(webApiPortStr + "/tcp"): []nat.PortBinding{
				{HostPort: webApiPortStr},
			},
		},
		Mounts: []mount.Mount{{
			Type:   mount.TypeBind,
			Source: common.GetEtcPath(),
			Target: "/itops/etc",
		}, {
			Type:   mount.TypeBind,
			Source: common.GetLwappPath(),
			Target: "/itops/nginx/html/lwjk_app",
		}, {
			Type:   mount.TypeBind,
			Source: common.GetPersistenceVolume(),
			Target: pvPath, // 数据持久化目录
		}, {
			Type:   mount.TypeBind,
			Source: common.GetDeploymentLogPath(),
			Target: "/itops/logs",
		}},
	}

	n := &network.NetworkingConfig{}

	currentPathEnvSuffix := GetCurrentPathEnvContainerSuffix()
	r, err := d.ContainerCreate(context.TODO(), c, h, n, nil, WebRunContainerName+currentPathEnvSuffix)
	if err != nil {
		diary.Errorf("创建容器失败：%v", err)
		return nil
	} else {
		diary.Infof("创建容器id：%v", r.ID)
	}

	return GetCurrentExistWebContainer()
}

// 启动web容器
func RunContainer() bool {
	webContainer := GetCurrentExistWebContainer()
	if webContainer == nil {
		diary.Errorf("未创建WEB容器，请先使用 lwctl build 命令创建容器！")
		return false
	}

	if strings.Contains(webContainer.Status, "Up") {
		diary.Errorf("WEB容器已经已经在运行中！")
		return false
	}

	if !strings.Contains(gogit.GetRepositoryCurrentBranch(common.GetEtcPath()), EtcBranchRuntimeSuffix) {
		diary.Errorf("配置包etc存在更新，请重新生成容器（lwctl build）")
		return false
	}

	d := docker.NewDockerClient()

	nginxStartStdoutFile := common.GetDeploymentLogPath() + "/nginx/nginx_stdout.log"
	os.Remove(nginxStartStdoutFile)
	err := d.ContainerStart(context.TODO(), webContainer.ID, types.ContainerStartOptions{})
	if err != nil {
		diary.Errorf("启动失败：容器启动失败：%v", err)
		return false
	}

	beginStart := time.Now().Unix()
	for {
		// 轮询检查nginx启动输出
		// 存在输出文件 或 5s超时 结束
		if util.FileExists(nginxStartStdoutFile) {
			time.Sleep(time.Millisecond * 100)
			break
		}
		time.Sleep(time.Millisecond * 100)
		if time.Now().Unix()-beginStart >= 5 {
			break
		}
	}

	out, err := os.ReadFile(nginxStartStdoutFile)
	if err != nil {
		fmt.Printf("WEB容器启动失败，没有nginx进程输出日志：%v\n", err)
		return false
	}

	if string(out) != "" {
		fmt.Printf("WEB容器进程nginx启动失败：%v \n", string(out))
		return false
	}

	fmt.Printf("容器启动成功，ID：%v \n", webContainer.ID)
	return true
}

// 获取当前部署目录唯一标识
func GetCurrentPathEnvContainerSuffix() string {
	currentEnvPath := common.GetLwopsVolume()
	suffix := strings.ReplaceAll(currentEnvPath, "/", "_")
	suffix = strings.ReplaceAll(suffix, "\\", "_")
	suffix = strings.ReplaceAll(suffix, ":", "_")
	return suffix
}

// 获取当前存在的web容器
func GetCurrentExistWebContainer() *types.Container {
	dock := docker.NewDockerClient()
	currentPathEnvSuffix := GetCurrentPathEnvContainerSuffix()

	currentEnvName := WebRunContainerName + currentPathEnvSuffix
	opt := filters.NewArgs(filters.KeyValuePair{Key: "name", Value: currentEnvName})
	list, err := dock.ContainerList(context.TODO(), types.ContainerListOptions{Filters: opt, All: true})
	if err != nil {
		fmt.Println("获取启动容器列表失败:", err)
		return nil
	}
	for _, c := range list {
		for _, nameItem := range c.Names {
			if strings.TrimPrefix(nameItem, "/") == currentEnvName {
				return &c
			}
		}
	}

	return nil
}

// 获取当前存在并运行的容器
func GetCurrentRunningWebContainer() *types.Container {
	webContainer := GetCurrentExistWebContainer()
	if webContainer == nil {
		return nil
	}
	if !strings.Contains(webContainer.Status, "Up") {
		return nil
	}
	return webContainer
}

// 停止启动的web容器
func StopContainer() bool {
	d := docker.NewDockerClient()
	webContainer := GetCurrentRunningWebContainer()
	if webContainer == nil {
		fmt.Println("没有容器在运行中！")
		return false
	}

	var timeOut time.Duration = 30

	err := d.ContainerStop(context.TODO(), webContainer.ID, &timeOut)
	if err == nil {
		fmt.Printf("关闭容器（%v）成功，ID:（%v）\n", webContainer.Names, webContainer.ID)
	} else {
		fmt.Printf("关闭容器（%v）失败，ID:（%v）\n", webContainer.Names, webContainer.ID)
	}

	return true
}

// 删除web容器
func RemoveWebContainer() bool {
	webContainer := GetCurrentExistWebContainer()
	if webContainer == nil {
		fmt.Println("未找到对应容器！")
		return false
	}

	if strings.Contains(webContainer.Status, "Up") {
		fmt.Printf("容器（%s）正在运行中，无法自动删除，请先手动停止容器！\n", webContainer.Names)
		return false
	}

	d := docker.NewDockerClient()
	err := d.ContainerRemove(context.TODO(), webContainer.ID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		fmt.Printf("删除容器（%v）失败：%v \n", webContainer.ID, err)
		return false
	} else {
		fmt.Println("容器删除成功：", webContainer.Names)
		return true
	}
}

// 获取部署目录使用的用户权限uid
func GetUseRunContainerUserUid() int {
	useUid := os.Getuid()
	if useUid == 0 {
		useUid = common.UseDefaultUserUid // 当前系统itops用户，uid不定
	}
	return useUid
}
