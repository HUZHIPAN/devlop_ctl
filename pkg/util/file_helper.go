package util

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
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
	err = filepath.Walk(dir,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}
			itemErr := os.Chown(path, uid, gid)
			if itemErr != nil && !os.IsNotExist(itemErr) {
				return itemErr
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
