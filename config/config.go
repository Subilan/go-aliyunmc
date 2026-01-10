package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

var Cfg Config

type Config struct {
	Base     BaseConfig     `toml:"base" validate:"required"`
	Aliyun   AliyunConfig   `toml:"aliyun" validate:"required"`
	Database DatabaseConfig `toml:"database" validate:"required"`
	Monitor  MonitorConfig  `toml:"monitor" validate:"required"`
	Deploy   DeployConfig   `toml:"deploy" validate:"required"`
	Server   ServerConfig   `toml:"server" validate:"required"`
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

func Load(filename string) {
	err := load(filename)

	if err != nil {
		log.Fatal(err)
	}
}

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

	err = validate.RegisterValidation("posRange", validPositiveRange)

	if err != nil {
		log.Println("cannot register posRange to validator:", err)
		return err
	}

	err = validate.Struct(Cfg)

	if err != nil {
		log.Println("config validation error:", err)
		return err
	}

	log.Print("OK")
	return nil
}
