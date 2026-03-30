package main

import (
	"fmt"
	"os"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/casbin"
	"go-aliyunmc/store"

	"github.com/pelletier/go-toml/v2"
)

func main() {
	// 创建configs目录
	configDir := "example_configs"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating configs directory: %v\n", err)
		os.Exit(1)
	}

	// 生成main.toml - 系统基础配置
	mainConfig := map[string]interface{}{
		"expose": 45678,
		"cors": map[string]interface{}{
			"allow_origins":     []string{"*"},
			"allow_credentials": true,
			"allow_headers":     []string{"Content-Length", "Content-Type", "Authorization", "Last-Event-Id"},
			"allow_methods":     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		},
		"autotls": map[string]interface{}{
			"enabled": false,
			"domains": []string{},
		},
		"session": map[string]interface{}{
			"key_pairs": []map[string]string{
				{
					"auth_key": "your-authentication-key-here",
					"enc_key":  "your-encryption-key-here",
				},
			},
		},
	}

	// 生成store.toml - 数据库配置
	storeConfig := store.Config{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "",
		DBName:   "aliyunmc",
		Charset:  "utf8mb4",
		SSLMode:  "disable",
		Path:     "",
	}

	// 生成casbin.toml - Casbin配置
	casbinConfig := casbin.Config{
		ModelPath:  "casbin/rbac_model.conf",
		PolicyPath: "casbin/rbac_policy.csv",
	}

	// 生成aliyun.toml - 阿里云配置
	aliyunConfig := aliyun.Config{
		RegionId:        "cn-hangzhou",
		AccessKeyId:     "your-access-key-id",
		AccessKeySecret: "your-access-key-secret",
		Ecs: aliyun.AliyunEcsConfig{
			InternetMaxBandwidthOut: 5,
			ImageId:                 "centos_7_9_x64_20G_alibase_20230619.vhd",
			SystemDisk: aliyun.EcsDiskConfig{
				Category: "cloud_essd",
				Size:     40,
			},
			DataDisk: aliyun.EcsDiskConfig{
				Category: "cloud_essd",
				Size:     100,
			},
			HostName:                 "aliyunmc-server",
			RootPassword:             "your-root-password",
			ProdPassword:             "your-prod-password",
			SpotInterruptionBehavior: "Stop",
			SecurityGroupId:          "your-security-group-id",
		},
	}

	// 写入main.toml
	if err := writeConfigFile(configDir+"/main.toml", mainConfig); err != nil {
		fmt.Printf("Error writing main.toml: %v\n", err)
		os.Exit(1)
	}

	// 写入store.toml
	if err := writeConfigFile(configDir+"/store.toml", storeConfig); err != nil {
		fmt.Printf("Error writing store.toml: %v\n", err)
		os.Exit(1)
	}

	// 写入casbin.toml
	if err := writeConfigFile(configDir+"/casbin.toml", casbinConfig); err != nil {
		fmt.Printf("Error writing casbin.toml: %v\n", err)
		os.Exit(1)
	}

	// 写入aliyun.toml
	if err := writeConfigFile(configDir+"/aliyun.toml", aliyunConfig); err != nil {
		fmt.Printf("Error writing aliyun.toml: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration files generated successfully in configs directory!")
}

// writeConfigFile 写入配置文件
func writeConfigFile(filePath string, config interface{}) error {
	out, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, out, 0644)
}
