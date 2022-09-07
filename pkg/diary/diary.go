package diary

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	flag       = log.Ldate | log.Ltime
	preDebug   = "[DEBUG]"
	preInfo    = "[INFO]"
	preWarning = "[WARNING]"
	preError   = "[ERROR]"
)

var (
	logFile        io.Writer
	debugLogger    *log.Logger
	infoLogger     *log.Logger
	warningLogger  *log.Logger
	errorLogger    *log.Logger
	ob_start       bool
	ob_buffer      string
	defaultLogFile = ""
)

var (
	IsRealTimeOutput bool = false // 是否实时输出审计
)

func init() {
	// loadLogFile()
}

// 设置日志目录
func SetLogPath(path string) {
	// /lwops/tmp/log/lwctl.log
	defaultLogFile = path + "/lwctl.log"
	if !triggerLogFileReset() {
		loadLogFile()
	}
}

func loadLogFile() {
	var err error
	os.MkdirAll(filepath.Dir(defaultLogFile), os.ModePerm)
	logFile, err = os.OpenFile(defaultLogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("create log file err： %+v", err)
	}
	debugLogger = log.New(logFile, preDebug, flag)
	infoLogger = log.New(logFile, preInfo, flag)
	warningLogger = log.New(logFile, preWarning, flag)
	errorLogger = log.New(logFile, preError, flag)
}

// 触发检查达到条件重置日志并返回true
func triggerLogFileReset() bool {
	if PeekFileSize(defaultLogFile) > 1024*1024*20 { // 文件大小达到20M
		bakFileName := time.Now().Format("2006-01-02_15_04_05") + "_lwctl.log"
		bakFilePath := filepath.Dir(defaultLogFile) + "/" + bakFileName
		_, err := copyFile(defaultLogFile, bakFilePath)
		if err != nil {
			Errorf("复制备份lwctl日志文件（%v）发生错误：%v", defaultLogFile, err)
			return false
		}
		err = os.Truncate(defaultLogFile, 0)
		if err != nil {
			Errorf("清空lwctl日志文件（%v）发生错误：%v", defaultLogFile, err)
		}
		loadLogFile()
		Infof("lwctl重置日志文件（%v）", defaultLogFile)
		return true
	}
	return false
}

// 检查日志文件，超出20M重置
func CheckLogFile() {
	if triggerLogFileReset() {
		Infof("lwctl serviced 定时检查触发了重置日志")
	} else {
		loadLogFile()
		Infof("lwctl serviced 定时检查重新加载日志文件")
	}
}

func copyFile(srcFile, destFile string) (int64, error) {
	file1, err := os.Open(srcFile)
	if err != nil {
		return 0, err
	}
	file2, err := os.OpenFile(destFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return 0, err
	}
	defer file1.Close()
	defer file2.Close()
	return io.Copy(file2, file1)
}

func Ob_Start() {
	if ob_start {
		ob_buffer = ""
	}
	ob_start = true
}

func Ob_get_contents() string {
	return ob_buffer
}

func GetGlobalLogContent() (string, error) {
	data, err := os.ReadFile(defaultLogFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Ob_End() {
	ob_start = false
}
func Ob_Clean() {
	ob_buffer = ""
}

func Ob_Printf(format string, v ...interface{}) {
	if ob_start {
		s := fmt.Sprintf(format, v...) + "\n"
		ob_buffer += s
	}

	if IsRealTimeOutput {
		fmt.Printf(format+"\n", v...)
	}
}

// 查看文件大小
func PeekFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		return -1
	}
	return fi.Size()
}

func Debugf(format string, v ...interface{}) {
	Ob_Printf(format, v...)
	debugLogger.Printf(format, v...)
}

func Infof(format string, v ...interface{}) {
	Ob_Printf(format, v...)
	infoLogger.Printf(format, v...)
}

func Warningf(format string, v ...interface{}) {
	Ob_Printf(format, v...)
	warningLogger.Printf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	Ob_Printf(format, v...)
	errorLogger.Printf(format, v...)
}

func SetOutputPath(path string) {
	var err error
	logFile, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("create log file err %+v", err)
	}
	debugLogger.SetOutput(logFile)
	infoLogger.SetOutput(logFile)
	warningLogger.SetOutput(logFile)
	errorLogger.SetOutput(logFile)
}
