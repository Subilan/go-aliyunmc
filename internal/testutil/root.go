package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ProjectRoot 返回仓库根目录，定位方式基于本文件路径，不依赖当前工作目录。
func ProjectRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("无法定位 testutil 源文件")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

// ChdirProjectRoot 将测试进程的工作目录切换到仓库根目录。
func ChdirProjectRoot() error {
	root, err := ProjectRoot()
	if err != nil {
		return err
	}
	return os.Chdir(root)
}
