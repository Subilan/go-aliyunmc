package main

import (
	"fmt"
	"os"

	"go-aliyunmc-v2/config"
	"go-aliyunmc-v2/store"

	"github.com/pelletier/go-toml/v2"
)

func main() {
	// 创建默认配置
	cfg := config.Config{
		Base: config.BaseConfig{
			Expose: 45678,
			Cors: config.CorsConfig{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				AllowHeaders:     []string{"Content-Length", "Content-Type", "Authorization", "Last-Event-Id"},
				AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			},
			Autotls: config.AutotlsConfig{
				Enabled: false,
			},
			Session: config.SessionConfig{
				KeyPairs: []config.SessionKeyPair{
					{
						AuthKey: "your-authentication-key-here",
						EncKey:  "your-encryption-key-here",
					},
				},
			},
		},
		Store: store.StoreConfig{
			Driver:   "mysql",
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "",
			DBName:   "aliyunmc",
			Charset:  "utf8mb4",
			SSLMode:  "disable",
			Path:     "",
		},
	}

	// 序列化配置为TOML
	out, err := toml.Marshal(&cfg)
	if err != nil {
		fmt.Printf("Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	// 写入文件
	err = os.WriteFile("../../config.example.toml", out, 0644)
	if err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("config.example.toml generated successfully!")
}
