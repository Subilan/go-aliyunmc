package config

import (
	"go-aliyunmc-v2/store"
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

var Global Config

type Config struct {
	// Base 是 go-aliyunmc 的基础配置
	Base  BaseConfig        `toml:"base" validate:"required"`
	// Store 是数据库配置
	Store store.StoreConfig `toml:"store" validate:"required"`
}

// MustBindGlobal 读取并解析全局配置，将结果绑定到 Global 变量上
func MustBindGlobal() {
	configContent, err := os.ReadFile("config.toml")

	if err != nil {
		log.Fatalln("找不到配置文件")
	}

	err = toml.Unmarshal(configContent, &Global)

	if err != nil {
		log.Fatalln("解析配置文件失败", err)
	}

	validate := validator.New()
	err = validate.Struct(Global)

	if err != nil {
		log.Fatalln("配置文件内容不符合要求", err)
	}
}

// TestDefault 返回默认配置用于测试
func TestDefault() Config {
	return Config{
		Base: BaseConfig{
			Expose: 45678,
			Cors: CorsConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				AllowHeaders:     []string{"Content-Length", "Content-Type", "Authorization", "Last-Event-Id"},
				AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			},
			Autotls: AutotlsConfig{
				Enabled: false,
			},
		},
		Store: store.StoreConfig{
			Driver: "sqlite",
			DBName: "test",
			Path:   ":memory:",
		},
	}
}