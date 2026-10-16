// Package utils 用于提供一些与具体实现无关的工具函数
package utils

import (
	"os"
	"path/filepath"

	"github.com/Subilan/go-aliyunmc/log_util"
)

// ProjectRoot 返回项目根目录的绝对路径，依赖于 go.mod 文件的存在。如果无法找到 go.mod 文件，程序将会终止。
// 这个函数只能用于开发环境。
func ProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		log_util.Fatal("无法获取当前工作目录")
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	log_util.Fatal("无法定位项目根目录")
	return ""
}
