package serviced

import (
	"fmt"
	"lwapp/pkg/diary"
	"lwapp/src/common"
	"lwapp/src/structure"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	HttpServerPort int = 8082
)

func ServicedCommandHandle(params *structure.ServicedParams) bool {
	ok := StartWebServer(params)
	if !ok {
		fmt.Println("服务启动失败！")
		return false
	}

	fmt.Println("服务启动成功！")
	return true
}

func StartWebServer(servicedParams *structure.ServicedParams) bool {
	// 设置日志驱动
	// log.SetOutput(&lf)

	HttpServerPort = servicedParams.Port

	// 路由初始化
	router := InitRouter()

	// http服务
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", HttpServerPort),
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}

	if servicedParams.Daemon {
		execProgram, err := os.Executable()
		if err != nil {
			fmt.Println("获取运行程序名称错误：", err)
			return false
		}

		execArgs := filterKeyword(os.Args[1:], "daemon")
		cmd := exec.Command(execProgram, execArgs...)

		servicedLogFile := common.GetTmpLogPath() + "/serviced.log"
		logFile, err := os.OpenFile(servicedLogFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			fmt.Println("打开serviced日志文件错误：", err)
		}

		cmd.Stdout = logFile
		if err := cmd.Start(); err != nil {
			fmt.Println("启动后台进程发生错误：", err)
		}

		fmt.Printf("服务进程PID：%v，服务端口：%d \n", cmd.Process.Pid, HttpServerPort)
		audit := fmt.Sprintf("启动命令：%v, 启动参数：%v \n", execProgram, execArgs)
		fmt.Print(audit)
		logFile.WriteString(audit)
		fmt.Printf("启动命令调用成功，请注意检查进程是否启动成功，日志文件：（%v）\n", servicedLogFile)
		os.Exit(0)
	}

	diary.Ob_End()
	diary.Ob_Clean()
	common.UnlockLwopsEnv()
	go runTimeCheck()

	fmt.Printf("[%v]\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("部署环境：%v \n", common.GetLwopsVolume())
	fmt.Printf("启动服务，开始监听端口：%v\n", HttpServerPort)
	err := s.ListenAndServe()
	fmt.Println("发生错误：", err)
	return false
}

// 定时检查
func runTimeCheck() {
	t := time.NewTicker(time.Hour * 2)

	for {
		<-t.C
		diary.CheckLogFile() // 检查日志文件
	}
}

// InitRouter 路由初始化
func InitRouter() *gin.Engine {

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(gin.Logger())
	// r.Use(gin.LoggerWithConfig(gin.LoggerConfig{Output: &logfile.LogFile{File_Path: "log.txt"}}))

	r.Use(gin.Recovery())

	apiv1 := r.Group("/lwctl/v1")
	{
		apiv1.POST("/upload-package", UploadPackage)
		apiv1.POST("/proxy-call", ProxyCall)
		apiv1.POST("/version-history", GetAppVersionList)
		apiv1.POST("/commit-diff", GetCommitChangeList)
		apiv1.POST("/seek-audit", SeekAudit)
	}

	return r
}

func filterKeyword(args []string, keyword string) []string {
	filterArgs := []string{}

	for _, val := range args {
		if strings.Contains(val, keyword) {
			continue
		}
		filterArgs = append(filterArgs, val)
	}
	return filterArgs
}
