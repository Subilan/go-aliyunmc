// Package config 包含系统完整的配置文件定义，按照功能模块进行划分。
package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

var Cfg Config

// Config 包含了整个系统的完整配置文件定义。
type Config struct {
	// Base 是系统的基础配置。
	Base BaseConfig `toml:"base" validate:"required"`

	// Aliyun 是与阿里云服务相关的配置。
	Aliyun AliyunConfig `toml:"aliyun" validate:"required"`

	// Database 是与本地 DBMS 相关的配置。
	Database DatabaseConfig `toml:"database" validate:"required"`

	// Monitor 是监控器相关配置。
	Monitor MonitorConfig `toml:"monitor" validate:"required"`

	// Deploy 是与服务器部署的相关配置。
	Deploy DeployConfig `toml:"deploy" validate:"required"`

	// Server 是与 Minecraft 服务器的相关配置。
	Server ServerConfig `toml:"server" validate:"required"`
}

func (c Config) GetAliyunEcsConfig() AliyunEcsConfig {
	return c.Aliyun.Ecs
}
func (c Config) GetGamePort() uint16 {
	return c.Server.Port
}
func (c Config) GetGameRconPort() uint16 {
	return c.Server.RconPort
}

// Load 用于完成配置文件内容的读取，如果出现了错误，此函数将导致程序退出。
func Load(filename string) {
	err := load(filename)

	if err != nil {
		log.Fatal(err)
	}
}

// load 用于读取配置文件内容并存入全局变量
func load(filename string) error {
	log.Print("Loading config...")

	configFileContent, err := os.ReadFile(filename)

	if err != nil {
		log.Println("Error reading config file:", err)
		return err
	}

	err = toml.Unmarshal(configFileContent, &Cfg)

	if err != nil {
		log.Println("cannot unmarshal config.toml:", err)
		return err
	}

	validate := validator.New()

	err = validate.Struct(Cfg)

	if err != nil {
		log.Println("config validation error:", err)
		return err
	}

	log.Print("OK")
	return nil
}
