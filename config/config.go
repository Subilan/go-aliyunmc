package config

import (
	"fmt"
	"path/filepath"
)

var Cfg Config

type ServerConfig struct {
	Expose    int    `toml:"expose"`
	JwtSecret string `toml:"jwt_secret"`
}

type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Database string `toml:"database"`
}

type DeployConfig struct {
	Packages     []string `toml:"packages"`
	SSHPublicKey string   `toml:"ssh_public_key"`
	JavaVersion  uint     `toml:"java_version"`
	OSSRoot      string   `toml:"oss_root"`
	BackupPath   string   `toml:"backup_path"`
	ArchivePath  string   `toml:"archive_path"`
}

func (d DeployConfig) BackupOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.BackupPath)
}

func (d DeployConfig) ArchiveOSSPath() string {
	return "oss://" + filepath.Join(d.OSSRoot[6:], d.ArchivePath)
}

type AliyunConfig struct {
	RegionId        string          `toml:"region_id"`
	AccessKeyId     string          `toml:"access_key_id"`
	AccessKeySecret string          `toml:"access_key_secret"`
	Ecs             AliyunEcsConfig `toml:"ecs"`
}

type AliyunEcsConfig struct {
	InternetMaxBandwidthOut  int           `toml:"internet_max_bandwidth_out"`
	ImageId                  string        `toml:"image_id"`
	SystemDisk               EcsDiskConfig `toml:"system_disk"`
	DataDisk                 EcsDiskConfig `toml:"data_disk"`
	HostName                 string        `toml:"hostname"`
	RootPassword             string        `toml:"root_password"`
	ProdPassword             string        `toml:"prod_password"`
	SpotInterruptionBehavior string        `toml:"spot_interruption_behavior"`
	SecurityGroupId          string        `toml:"security_group_id"`
}

// EcsEndpoint 返回 ECS 相关的请求 endpoint
func (c AliyunConfig) EcsEndpoint() string {
	return fmt.Sprintf("ecs.%s.aliyuncs.com", c.RegionId)
}

// VpcEndpoint 返回 VPC 相关请求 endpoint
func (c AliyunConfig) VpcEndpoint() string {
	return fmt.Sprintf("vpc.%s.aliyuncs.com", c.RegionId)
}

type EcsDiskConfig struct {
	Category string `toml:"category"`
	Size     int    `toml:"size"`
}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Aliyun   AliyunConfig   `toml:"aliyun"`
	Database DatabaseConfig `toml:"database"`
	Monitor  MonitorConfig  `toml:"monitor"`
	Deploy   DeployConfig   `toml:"deploy"`
}

func (c Config) GetAliyunEcsConfig() AliyunEcsConfig {
	return c.Aliyun.Ecs
}

type MonitorConfig struct {
	ActiveInstanceStatusMonitor ActiveInstanceStatusMonitor `toml:"active_instance_status"`
	AutomaticPublicIpAllocator  AutomaticPublicIpAllocator  `toml:"automatic_public_ip_allocator"`
}

type ActiveInstanceStatusMonitor struct {
	ExecutionInterval int  `toml:"execution_interval"`
	Verbose           bool `toml:"verbose"`
}

type AutomaticPublicIpAllocator struct {
	ExecutionInterval int  `toml:"execution_interval"`
	Verbose           bool `toml:"verbose"`
}
