// Package utils 用于提供一些与具体实现无关的工具函数
package utils

import (
	"go-aliyunmc/log_util"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

const ConfigDir = "configs/"

// MustBindConfig 读取并绑定 ConfigDir 下的 toml 格式配置文件到指定对象，遇到任何错误都会导致程序终止。
func MustBindConfig[T any](obj *T, name string) {
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
