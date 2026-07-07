// Package utils 用于提供一些与具体实现无关的工具函数
package utils

import (
	"github.com/Subilan/go-aliyunmc/log_util"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

const ConfigDir = "configs/"

// MustBindConfigToml 读取并绑定 ConfigDir 下的 toml 格式配置文件到指定对象，遇到任何错误都会导致程序终止。
func MustBindConfigToml[T any](obj *T, name string) {
	if os.Getenv("CONFIGGEN_SKIP_CONFIG") == "1" {
		return
	}
	path := ConfigDir + name + ".toml"
	fileContent, err := os.ReadFile(path)

	if err != nil {
		log_util.Fatal("无法读取配置文件" + path)
	}

	err = toml.Unmarshal(fileContent, obj)

	if err != nil {
		log_util.Fatal("无法解析配置文件")
	}

	validator := validator.New()
	err = validator.Struct(obj)

	if err != nil {
		log_util.Fatal("配置文件不合要求：" + err.Error())
	}
}

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

func init() {
	testing := strings.HasSuffix(os.Args[0], ".test")
	if testing {
		if err := os.Chdir(ProjectRoot()); err != nil {
			log_util.Fatal("切换到项目根目录失败: %v", err)
		}
	}
}
