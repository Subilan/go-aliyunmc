package config

import "fmt"

type AliyunConfig struct {
	RegionId        string          `toml:"region_id" validate:"required"`
	AccessKeyId     string          `toml:"access_key_id" validate:"required"`
	AccessKeySecret string          `toml:"access_key_secret" validate:"required"`
	Ecs             AliyunEcsConfig `toml:"ecs" validate:"required"`
}

type AliyunEcsConfig struct {
	InternetMaxBandwidthOut  int           `toml:"internet_max_bandwidth_out" validate:"required,max=100"`
	ImageId                  string        `toml:"image_id" validate:"required"`
	SystemDisk               EcsDiskConfig `toml:"system_disk" validate:"required"`
	DataDisk                 EcsDiskConfig `toml:"data_disk" validate:"required"`
	HostName                 string        `toml:"hostname" validate:"required"`
	RootPassword             string        `toml:"root_password" validate:"required"`
	ProdPassword             string        `toml:"prod_password" validate:"required"`
	SpotInterruptionBehavior string        `toml:"spot_interruption_behavior" validate:"required,oneof=Stop Terminate"`
	SecurityGroupId          string        `toml:"security_group_id" validate:"required"`
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
	Category string `toml:"category" validate:"required"`
	Size     int    `toml:"size" validate:"required,min=20"`
}
