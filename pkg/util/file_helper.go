package util

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// 文件或目录是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// 判断路径是否存在
func IsExists(path string) (os.FileInfo, bool) {
	f, err := os.Stat(path)
	return f, err == nil || os.IsExist(err)
}

// 获取目录下的列表（一层）
func GetDirFileList(dirPath string) []fs.FileInfo {
    f,err := os.OpenFile(dirPath, os.O_RDONLY,os.ModeDir)
    if err != nil {
		return []fs.FileInfo{}
	}
	defer f.Close()
	
	//读取目录项
	fileList,err := f.Readdir(-1) //-1读取目录中的所有目录项
	if err != nil{
		return []fs.FileInfo{}
	}
	return fileList
}

// 判断所给路径是否为文件夹
func IsDir(path string) (os.FileInfo, bool) {
	f, flag := IsExists(path)
	return f, flag && f.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) (os.FileInfo, bool) {
	f, flag := IsExists(path)
	return f, flag && !f.IsDir()
}

func WriteFileContent(filePath string, content string) (bool, error) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return false, err
	}
	defer file.Close()

	//写入时，使用带缓存的 *Writer
	writer := bufio.NewWriter(file)

	writer.WriteString(content)

	err = writer.Flush()
	if err != nil {
		return false, err
	}
	return true, nil
}

// 写入文件 目录不存在则创建（覆盖写入）
func WriteFileWithDir(filename string, content string) (bool, error) {
	path := filepath.Dir(filename)
	name := filepath.Base(filename)

	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return false, err
	}
	file, err := os.OpenFile(path+"/"+name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return false, err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	if err != nil {
		return false, err
	}
	return true, nil
}

// 递归改变目录权限
func ChownAll(dirPath string, uid, gid int) error {
	dir, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return err
		}
		itemErr := os.Chown(path, uid, gid)
		if itemErr != nil && !os.IsNotExist(itemErr) {
			return itemErr
		} else {
			os.Lchown(path, uid, gid)
		}
		return nil
	})
	return err
}

// 递归改变目录权限
func ChmodAll(dirPath string, mode fs.FileMode) error {
	dir, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}
	err = filepath.Walk(dir,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			os.Chmod(path, mode)
			return nil
		})
	return err
}

// 查看文件大小
func PeekFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		return -1
	}

	return fi.Size()
}

// 复制目录
func CopyDirectoryAll(srcPath string, destPath string, ignorePrefix, ignoreSuffix []string) error {
	srcPath, _ = filepath.Abs(srcPath)
	destPath, _ = filepath.Abs(destPath)

	if srcPath == "" || destPath == "" {
		return fmt.Errorf("无效的路径")
	}

	// 转换平台目录分隔符
	for i, ignoreItem := range ignorePrefix {
		ignorePrefix[i] = filepath.FromSlash(ignoreItem)
	}
	for i, ignoreItem := range ignoreSuffix {
		ignoreSuffix[i] = filepath.FromSlash(ignoreItem)
	}

	err := filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
		if d == nil {
			return err
		}

		relativePath := strings.TrimPrefix(path, srcPath)

		isIgnore := false
		for _, ignoreItem := range ignorePrefix { // 忽略前缀
			if strings.HasPrefix(relativePath, ignoreItem) {
				isIgnore = true
				break
			}
		}
		if !isIgnore {
			for _, ignoreItem := range ignoreSuffix { // 忽略后缀
				if strings.HasSuffix(relativePath, ignoreItem) {
					isIgnore = true
					break
				}
			}
		}
		// 忽略path包含指定字符的文件或目录，忽略
		if isIgnore {
			return nil
		}

		currentDestPath := destPath + relativePath
		info, err := d.Info()

		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 { // 判断是软链接
			return nil
		}

		if d.IsDir() {
			return os.MkdirAll(currentDestPath, info.Mode().Perm())
		} else {
			if currentDir := filepath.Dir(currentDestPath); !FileExists(currentDir) {
				os.MkdirAll(currentDir, os.ModePerm)
			}
			return CopyFile(path, currentDestPath, info.Mode().Perm())
		}
	})

	return err
}

// 复制一个文件
func CopyFile(srcFile, destFile string, perm fs.FileMode) error {
	fo, err := os.OpenFile(destFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer fo.Close()
	content, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}
	_, err = fo.Write(content)
	return err
}
