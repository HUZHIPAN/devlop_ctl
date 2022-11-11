package common

import (
	"fmt"
	"lwapp/pkg/util"
	"os"
	"path/filepath"
	"strconv"
)

// 对一个部署目录进行加锁（同一个环境的操作互斥）
// 通过对部署目录写入一个锁文件lock.pid文件
// 当多个进程尝试设置同一个部署目录时，
// 存在一个锁文件且pid对应进程未关闭，视为持有锁
// 等待锁释放，再将当前进程pid写入锁文件
func LockLwopsEnv() error {
	lockFile := getLockFile()
	content, _ := os.ReadFile(lockFile)

	var existLock bool
	existPID, err := strconv.Atoi(string(content))
	if err != nil || existPID <= 0 {
		existLock = false
	} else {
		execute, _ := os.Executable()
		existLock = util.CheckProcessIsRunning(existPID, filepath.Base(execute))
	}

	if existLock {
		return fmt.Errorf("另一个进程持有部署目录操作锁，进程pid：%v", string(content))
	}

	_, err = util.WriteFileWithDir(lockFile, strconv.Itoa(os.Getpid()))
	if err != nil {
		return fmt.Errorf("无法写入锁文件（%v），error：%v", lockFile, err)
	} 

	return nil
}

// 释放一个部署目录的持有锁
// 主动调用解锁 / 进程退出 视为释放锁
func UnlockLwopsEnv() error {
	err := os.Remove(getLockFile())
	if err != nil {
		return fmt.Errorf("删除锁文件（%v）失败：%v", getLockFile(), err)
	}
	return nil
}

func getLockFile() string {
	return GetTmpPath() + "/lock.pid"
}
