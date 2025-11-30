package config

import (
	"fmt"
)

var Cfg Config

type ServerConfig struct {
	Expose    int    `toml:"expose"`
	JwtSecret string `toml:"jwt_secret"`
}

type AliyunConfig struct {
	AccessKeyId     string          `toml:"access_key_id"`
	AccessKeySecret string          `toml:"access_key_secret"`
	Ecs             AliyunEcsConfig `toml:"ecs"`
}

type AliyunEcsConfig struct {
	RegionId                 string        `toml:"region_id"`
	InternetMaxBandwidthOut  int           `toml:"internet_max_bandwidth_out"`
	ImageId                  string        `toml:"image_id"`
	SystemDisk               EcsDiskConfig `toml:"system_disk"`
	DataDisk                 EcsDiskConfig `toml:"data_disk"`
	HostName                 string        `toml:"hostname"`
	Password                 string        `toml:"password"`
	SpotInterruptionBehavior string        `toml:"spot_interruption_behavior"`
	SecurityGroupId          string        `toml:"security_group_id"`
}

func (c AliyunEcsConfig) Endpoint() string {
	return fmt.Sprintf("ecs.%s.aliyuncs.com", c.RegionId)
}

type EcsDiskConfig struct {
	Category string `toml:"category"`
	Size     int    `toml:"size"`
}

type Config struct {
	Server ServerConfig `toml:"server"`
	Aliyun AliyunConfig `toml:"aliyun"`
}

func (c Config) GetAliyunEcsConfig() AliyunEcsConfig {
	return c.Aliyun.Ecs
}
