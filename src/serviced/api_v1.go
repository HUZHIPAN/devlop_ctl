package serviced

import (
	"lwapp/pkg/diary"
	"lwapp/pkg/gogit"
	"lwapp/src/common"
	"lwapp/src/parse"
	"os"
	"os/exec"
	"strings"
	"sync"

	"net/http"
	"path"

	"github.com/gin-gonic/gin"
)

// 上传更新包
func UploadPackage(c *gin.Context) {
	//从请求中读取文件
	file, err := c.FormFile("file") //请求中获取携带的参数,就是html文件中的name="f1"
	if err != nil {                 //读取失败，将错误报出来
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	} else { //读取成功，就保存到服务端本地
		fileDest := path.Join(parse.GetRequestPackagePath(), file.Filename)
		err := c.SaveUploadedFile(file, fileDest)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   true,
			"msg":      "",
			"filepath": fileDest,
		})
	}
}

// 执行命令
func ProxyCall(c *gin.Context) {
	argsParam := c.PostForm("args")
	args := strings.Fields(argsParam)

	execProgram, err := os.Executable()
	if err != nil {
		c.JSON(500, gin.H{
			"status": false,
			"msg":    err.Error(),
		})
		return
	}

	cmd := exec.Command(execProgram, args...)
	resultByte, err := cmd.Output()
	if err != nil {
		c.JSON(500, gin.H{
			"status": false,
			"msg":    err.Error(),
		})
		return
	}
	result := string(resultByte)

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"msg":    "",
		"audit":  result,
	})
}

// 查看全部审计日志
func SeekAudit(c *gin.Context) {
	logContent, err := diary.GetGlobalLogContent()
	if err != nil {
		diary.Errorf("获取全局日志发生异常：%v", err)
		c.JSON(500, gin.H{
			"status":  false,
			"content": logContent,
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"content": logContent,
		"msg":     "",
	})
}

// 获取更新版本记录
func GetAppVersionList(c *gin.Context) {
	lwappPath := common.GetLwappPath()
	currentBranch := gogit.GetRepositoryCurrentBranch(lwappPath)
	branchList := gogit.GetRepositoryBranchList(common.GetLwappPath())

	lock := sync.Mutex{}
	list := []map[string]interface{}{}

	wg := sync.WaitGroup{}
	wg.Add(len(branchList))
	for _, branchName := range branchList {
		go getBranchBaseInfo(lwappPath, branchName, &list, &wg, &lock)
	}
	wg.Wait()
	c.JSON(http.StatusOK, gin.H{
		"status":        true,
		"currentBranch": currentBranch,
		"list":          list,
		"msg":           "",
	})
}

func getBranchBaseInfo(lwappPath, branchName string, list *[]map[string]interface{}, wg *sync.WaitGroup, lock *sync.Mutex) {
	commitHistory := gogit.GetBranchCommitInfo(lwappPath, branchName)
	versionInfo := common.GetAppVersionNumberInfo(branchName)

	result := map[string]interface{}{
		"branch":        branchName,
		"version":       versionInfo.VersionNumber,
		"updateType":    versionInfo.Mode,
		"commitHistory": commitHistory,
	}

	lock.Lock()
	*list = append(*list, result)
	lock.Unlock()
	wg.Done()
}

// 获取更新版本记录
func GetCommitChangeList(c *gin.Context) {
	commitHash := c.PostForm("commitHash")
	lwappPath := common.GetLwappPath()

	changeList, err := gogit.GetCommitChangeList(lwappPath, commitHash)
	if changeList == nil || err != nil {
		c.JSON(200, gin.H{
			"status": false,
			"list":   nil,
			"msg":    "获取失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"list":   changeList,
		"total":  len(*changeList),
		"msg":    "获取成功",
	})
}
