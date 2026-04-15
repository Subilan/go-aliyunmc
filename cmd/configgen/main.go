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

	monitorFileSyncConfig := monitors.FileSyncMonitorConfig{
		LocalCacheRoot: "remote_data_cache",
		Files: []monitors.FileConfig{
			{
				RemotePath:      "/path/to/remote/file1.log",
				LocalPath:       "logs/file1.log",
				PollIntervalSec: 10,
			},
		},
	}

	monitorAutoArchiveIdleConfig := monitors.AutoArchiveIdleMonitorConfig{
		Enabled:                 false,
		IdleCountdownSec:        1200,
		StopWaitTimeoutSec:      60,
		OfflineCheckIntervalSec: 3,
		DeleteIgnoreNonExistent: true,
	}

	monitorBackupConfig := monitors.BackupMonitorConfig{
		Enabled:         false,
		PollIntervalSec: 600,
	}

	monitorBestEcsCandidateConfig := monitors.BestEcsCandidateMonitorConfig{
		PollIntervalSec:     600,
		MemChoices:          []int{8, 16},
		CpuCoreCountChoices: []int{4, 8},
		CacheFile:           "ecs_candidates.json",
		Filters: monitors.InstanceChargeFilters{
			MaxTradePrice:         0.6,
			InstanceTypeExclusion: "^ecs\\.(e|s6|xn4|n4|mn4|e4|t|d).*$",
		},
	}

	deployTaskConfig := tasks.DeployTaskConfig{
		Exclusive: true,
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
		Vars: tasks.BackupTemplateVars{
			BackupOSSPath: "oss://your-bucket/backup",
			MaxKeepCount:  5,
			BaseDir:       "/home/mc/server/archive",
			TargetDirs:    []string{"world", "world_nether", "world_the_end"},
		},
	}

	archiveTaskConfig := tasks.ArchiveTaskConfig{
		TemplatePath: "scripts/archive.tmpl.sh",
		TimeoutSec:   1800,
		Exclusive:    true,
		Vars: tasks.ArchiveTemplateVars{
			OSSRoot:          "oss://your-bucket",
			ArchiveName:      "archive",
			RemoteArchiveDir: "/home/mc/server/archive",
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

	if err := writeConfigFile(configDir+"/monitor-file-sync.toml", monitorFileSyncConfig); err != nil {
		fmt.Printf("Error writing monitor-file-sync.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/monitor-auto-archive-idle.toml", monitorAutoArchiveIdleConfig); err != nil {
		fmt.Printf("Error writing monitor-auto-archive-idle.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/monitor-backup.toml", monitorBackupConfig); err != nil {
		fmt.Printf("Error writing monitor-backup.toml: %v\n", err)
		os.Exit(1)
	}

	if err := writeConfigFile(configDir+"/monitor-best-ecs-candidate.toml", monitorBestEcsCandidateConfig); err != nil {
		fmt.Printf("Error writing monitor-best-ecs-candidate.toml: %v\n", err)
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
