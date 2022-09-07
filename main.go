package main

import (
	"flag"
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/pkg/util"
	"lwapp/src/common"
	"lwapp/src/parse"
	"lwapp/src/serviced"
	"lwapp/src/structure"
	"os"
)

var (
	version string = "v1.0" // 工具版本
)

func main() {
	globalParams := &structure.GlobalParams{}

	applyParams := &structure.ApplyParams{}
	applyCmd := flag.NewFlagSet("apply", flag.ExitOnError)
	applyCmd.StringVar(&applyParams.PackagePath, "f", "", "更新包（只支持.tar.gz）")
	applyCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")

	webParams := &structure.WebParams{}
	webCmd := flag.NewFlagSet("web", flag.ExitOnError)
	webCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	webCmd.StringVar(&webParams.Action, "s", "status", "操作（start|stop|restart|status）")
	webCmd.BoolVar(&webParams.WithRemove, "rm", false, "删除容器，仅支持stop时指定，如：lwctl web -s stop -rm")

	buildParams := &structure.BuildParams{}
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	buildCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	buildCmd.IntVar(&buildParams.WebPort, "web-port", common.DefaultWebPort, "WEB前端端口")
	buildCmd.IntVar(&buildParams.WebApiPort, "web-api-port", common.DefaultWebApiPort, "WEB后端服务端口")
	buildCmd.StringVar(&buildParams.WebApiGateway, "web-api-gateway", "/backend_api", "配置后端服务访问的地址如（http://127.0.0.1:8081）")
	buildCmd.StringVar(&buildParams.MacAddr, "mac-addr", "", "容器内绑定的网卡mac地址")

	rollbackParams := &structure.RollbackParams{}
	rollbackCmd := flag.NewFlagSet("rollback", flag.ExitOnError)
	rollbackCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	rollbackCmd.StringVar(&rollbackParams.Type, "t", "", "回滚类型（image")

	appParams := &structure.AppParams{}
	appCmd := flag.NewFlagSet("app", flag.ExitOnError)
	appCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	appCmd.BoolVar(&appParams.ShowVersionList, "l", true, "查看部署的版本列表  示例：lwctl app -l")
	appCmd.StringVar(&appParams.ToVersion, "c", "", "切换版本号（v6.0.1） 示例：lwctl app -v v6.0.1")

	etcParams := &structure.EtcParams{}
	etcCmd := flag.NewFlagSet("etc", flag.ExitOnError)
	etcCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	etcCmd.BoolVar(&etcParams.ShowVersionList, "l", true, "查看配置包的版本列表  示例：lwctl etc -l")
	etcCmd.StringVar(&etcParams.ToVersion, "c", "", "切换配置包版本号（v0.11） 示例：lwctl etc -c v0.11")

	execParams := &structure.ExecParams{}
	execCmd := flag.NewFlagSet("exec", flag.ExitOnError)
	execCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	execCmd.StringVar(&execParams.Command, "c", "", "容器内执行的bash命令，默认工作目录（/itops/nginx/html/lwjk_app）")

	servicedParams := &structure.ServicedParams{}
	servicedCmd := flag.NewFlagSet("serviced", flag.ExitOnError)
	servicedCmd.StringVar(&globalParams.LwopsPath, "p", common.DefaultLwopsPath, "部署目录")
	servicedCmd.IntVar(&servicedParams.Port, "P", 8082, "服务端口")
	servicedCmd.BoolVar(&servicedParams.Daemon, "daemon", false, "以守护进程运行")

	if len(os.Args) < 2 || !util.InArray(os.Args[1], []string{"apply", "web", "rollback", "app", "etc", "exec", "serviced", "build"}) || util.InArray(os.Args[1], []string{"-h", "--help"}) {
		printHelp()
		return
	}

	switch os.Args[1] {
	case "apply":
		if len(os.Args) > 2 {
			applyCmd.Parse(os.Args[2:])
		} else {
			applyCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.ApplyCommandHandle(applyParams)

	case "build":
		if len(os.Args) >= 2 {
			buildCmd.Parse(os.Args[2:])
			if !enterBefore(globalParams) {
				return
			}
			if parse.IsDefaultBuildParams(buildParams) && parse.IsExistDeployEnvSetting() {
				fmt.Println("未指定生成容器配置，默认使用上一次生成容器使用的配置！")
				parse.BuildContainerByEnvParams()
				return
			}
		} else {
			buildCmd.Parse([]string{"-h"})
		}

		parse.BuildCommandHandle(buildParams)

	case "web":
		if len(os.Args) > 2 {
			webCmd.Parse(os.Args[2:])
		} else {
			webCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.WebCommandHandle(webParams)

	case "rollback":
		if len(os.Args) > 2 {
			rollbackCmd.Parse(os.Args[2:])
		} else {
			rollbackCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.RollbackCommandHandle(rollbackParams)

	case "etc":
		if len(os.Args) > 2 {
			etcCmd.Parse(os.Args[2:])
		} else {
			etcCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.EtcCommandHandle(etcParams)

	case "app":
		if len(os.Args) > 2 {
			appCmd.Parse(os.Args[2:])
		} else {
			appCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.AppCommandHandle(appParams)
	case "exec":
		if len(os.Args) > 2 {
			execCmd.Parse(os.Args[2:])
		} else {
			execCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		parse.ExecCommandHandle(execParams)
	case "serviced":
		if len(os.Args) >= 2 {
			servicedCmd.Parse(os.Args[2:])
		} else {
			servicedCmd.Parse([]string{"-h"})
		}
		if !enterBefore(globalParams) {
			return
		}
		serviced.ServicedCommandHandle(servicedParams)
	}
}

func enterBefore(globalParams *structure.GlobalParams) bool {
	ok := common.SetLwopsVolume(globalParams.LwopsPath) // 部署目录默认 /lwops
	if !ok {
		fmt.Println("部署路径异常！")
		return false
	}
	diary.Ob_Start()

	if os.Args[1] == "apply" {
		diary.IsRealTimeOutput = true // 实时输出审计
	}

	ok = common.ExecuteBeforeCheckHandle() // 执行前检查
	if !ok {
		fmt.Println(diary.Ob_get_contents())
		fmt.Println("环境监测异常！")
		return false
	}
	diary.Infof("执行%v操作：%v", os.Args[1], os.Args)
	return true
}

func printHelp() {
	desc := `
                    | |\ \      / / _ \|  _ \/ ___| 
                    | | \ \ /\ / / | | | |_) \___ \ 
                    | |__\ V  V /| |_| |  __/ ___) |
                    |_____\_/\_/  \___/|_|   |____/ 

lwctl工具针对乐维web的部署环境初始化及持续可靠的版本更新支持，需要当前系统装好docker环境
使用场景：
     1、生产环境快速构建（部署）
          拿到产品部署包，使用lwctl工具快速构建环境（仅支持tar.gz格式包）
               示例 ：lwctl apply -f lwjk_deploy_6.1.tar.gz 

     2、启动WEB服务
          2.1 生成运行容器，用于指定对外端口，指定容器内的物理地址（MAC地址，授权机器）
               示例：lwctl build --mac-addr=00-15-5D-A0-76-41 （支持WEB端口指定，使用默认端口可省略）
          2.2 管理服务（查看状态：status、启动：start、停止：stop、重启：restart）
               示例：lwctl web -s start

     3、WEB系统更新（支持镜像、环境配置、产品版本等更新）
          拿到对应的更新包，快速应用更新 （镜像或环境配置更新时，执行场景2.1的操作（自动完成））
               示例：lwctl apply -f lwjk_upgrade_6.1.1.tar.gz

     4、部署多个环境或指定部署目录（支持多个WEB环境并存，支持同时启动）
          支持在同一系统上部署多套WEB服务，-p为全局参数，指定部署目录，未指定默认使用/itops（推荐）
          不同部署目录之间相互独立，lwctl仅会操作-p指定目录的资源，不同部署目录可用于区分多个WEB环境
               示例：lwctl apply -f lwjk_package.tar.gz -p /itops2 （表示lwctl操作的是/itops2目录下的WEB环境）

     5、在web服务的:8081/dev工具界面操作更新
          在/dev工具操作版本更新，需要先开启lwctl的服务提供操作接口
               示例：lwctl serviced --daemon

工具版本：%v
其他维护或操作可查看下列支持的命令及子命令（可使用 lwctl 子命令 --help 获取使用帮助）：
	`

	fmt.Printf(desc, version)
	f := "%-10s %-30s %-40s\n"
	fmt.Println("\n", "commands：")
	fmt.Printf(f, "apply", "应用包更新（部署或更新）", "示例：lwctl apply -f package.tar.gz")
	fmt.Printf(f, "build", "创建运行容器，配置端口、绑定mac地址等", "示例：lwctl build --help")
	fmt.Printf(f, "app", "管理产品版本", "示例：lwctl app -c v6.0.1")
	fmt.Printf(f, "web", "管理WEB服务的启动停止", "示例：lwctl web --help")
	fmt.Printf(f, "etc", "管理配置包版本", "示例：lwctl etc --help")
	fmt.Printf(f, "exec", "在运行的WEB容器内执行命令", "示例：lwctl exec -c 'php bin/manager init'")
	fmt.Printf(f, "serviced", "启动http服务接收更新任务", "示例：lwctl serviced -daemon -P 8082")
	// fmt.Printf(f, "rollback", "按更新类型回滚最近一次更新", "示例：lwctl rollback -t image")
	fmt.Println("")
}
