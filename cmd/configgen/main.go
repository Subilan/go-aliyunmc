package main

import (
	"fmt"
	"os"

	"go-aliyunmc/aliyun"
	"go-aliyunmc/casbin"
	"go-aliyunmc/monitors"
	"go-aliyunmc/server"
	"go-aliyunmc/store"
	"go-aliyunmc/tasks"

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

	serverConfig := server.ServerConfig{
		Port:         25565,
		RconPort:     25575,
		RconPassword: "your-rcon-password",
	}

	monitorServerConfig := monitors.ServerStatusMonitorConfig{
		PollIntervalSec: 5,
	}

	monitorInstanceConfig := monitors.InstanceStatusMonitorConfig{
		PollIntervalSec: 5,
	}

	deployTaskConfig := tasks.DeployTaskConfig{
		Exclusive: true,
		SSH: tasks.TaskSSHConfig{
			ConnectTimeoutSec: 10,
		},
		Steps: []tasks.DeployStepConfig{
			{
				Name:       "创建用户",
				ScriptPath: "scripts/deploy.create-user.tmpl.sh",
				TimeoutSec: 30,
			},
			{
				Name:       "配置SSH授权",
				ScriptPath: "scripts/deploy.setup-ssh-authorized-keys.tmpl.sh",
				TimeoutSec: 30,
			},
			{
				Name:       "配置软件源",
				ScriptPath: "scripts/deploy.configure-apt-sources.tmpl.sh",
				TimeoutSec: 120,
			},
			{
				Name:       "配置Java仓库",
				ScriptPath: "scripts/deploy.setup-java-repo.tmpl.sh",
				TimeoutSec: 120,
			},
			{
				Name:       "安装系统软件",
				ScriptPath: "scripts/deploy.install-system-packages.tmpl.sh",
				TimeoutSec: 180,
			},
			{
				Name:       "安装ossutil",
				ScriptPath: "scripts/deploy.install-ossutil.tmpl.sh",
				TimeoutSec: 60,
			},
			{
				Name:       "挂载数据盘",
				ScriptPath: "scripts/deploy.format-and-mount-data-disk.tmpl.sh",
				TimeoutSec: 60,
			},
			{
				Name:       "恢复归档数据",
				ScriptPath: "scripts/deploy.restore-archive-data.tmpl.sh",
				TimeoutSec: 420,
			},
		},
		Vars: tasks.DeployTemplateVars{
			SSHPublicKey:   "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ...",
			JavaVersion:    21,
			Packages:       []string{"zip", "unzip"},
			ArchiveOSSPath: "oss://your-bucket/archive",
		},
	}

	backupTaskConfig := tasks.BackupTaskConfig{
		TemplatePath: "scripts/backup.tmpl.sh",
		TimeoutSec:   1800,
		Exclusive:    true,
		SSH: tasks.TaskSSHConfig{
			ConnectTimeoutSec: 10,
		},
		Vars: tasks.BackupTemplateVars{
			BackupOSSPath: "oss://your-bucket/backup",
		},
	}

	archiveTaskConfig := tasks.ArchiveTaskConfig{
		TemplatePath: "scripts/archive.tmpl.sh",
		TimeoutSec:   1800,
		Exclusive:    true,
		SSH: tasks.TaskSSHConfig{
			ConnectTimeoutSec: 10,
		},
		Vars: tasks.ArchiveTemplateVars{
			OSSRoot: "oss://your-bucket",
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

	if err := writeConfigFile(configDir+"/server.toml", serverConfig); err != nil {
		fmt.Printf("Error writing server.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/monitor-server.toml", monitorServerConfig); err != nil {
		fmt.Printf("Error writing monitor-server.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/monitor-instance.toml", monitorInstanceConfig); err != nil {
		fmt.Printf("Error writing monitor-instance.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/task-deploy.toml", deployTaskConfig); err != nil {
		fmt.Printf("Error writing task-deploy.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/task-backup.toml", backupTaskConfig); err != nil {
		fmt.Printf("Error writing task-backup.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/task-archive.toml", archiveTaskConfig); err != nil {
		fmt.Printf("Error writing task-archive.toml: %v\n", err)
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
