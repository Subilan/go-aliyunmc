package config

import "fmt"

// Aliyun 对应 [aliyun]：与阿里云相关的全部配置。
type Aliyun struct {
	// RegionId 是实例所在的地域，决定实例与 OSS 的内网连通性。
	RegionId string `toml:"region_id,commented" validate:"required" comment:"实例运行的地域，例如 cn-hangzhou。地域将影响实例与OSS之间的内网连通性，不同地域之间抢占式实例的库存情况也有所不同。"`

	// AccessKeyId 是系统访问阿里云服务的 AKID
	AccessKeyId string `toml:"access_key_id,commented" validate:"required" comment:"用于访问阿里云服务的AKID，可在阿里云RAM控制台中生成"`

	// AccessKeySecret 是系统访问阿里云服务器的 AK 密钥
	AccessKeySecret string `toml:"access_key_secret,commented" validate:"required" comment:"用于访问阿里云服务的AKSecret"`

	// Ecs 包含了对云服务器 ECS 服务的相关配置
	Ecs AliyunEcs `toml:"ecs"`
}

// AliyunEcs 对应 [aliyun.ecs]，包含了所有与云服务器 ECS 相关的配置内容。参考 https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
type AliyunEcs struct {
	// InternetMaxBandwidthOut 是实例的峰值带宽，单位 Mbps，取值范围 1～100 且为整数
	InternetMaxBandwidthOut int `toml:"internet_max_bandwidth_out" default:"5" validate:"min=1,max=100" comment:"实例的峰值带宽，单位为Mbps，取值范围1～100，必须为整数"`

	// ImageId 是实例使用的系统镜像名称，可以认为决定了操作系统的类型。
	// 取值可以参考 https://help.aliyun.com/zh/ecs/user-guide/find-an-image https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
	ImageId string `toml:"image_id,commented" validate:"required" comment:"实例使用的系统镜像名，决定操作系统类型"`

	// SystemDisk 是该实例使用的系统盘配置。
	SystemDisk AliyunDisk `toml:"system_disk"`

	// DataDisk 是该实例使用的第一个数据盘配置。一般不认为需要使用第二个数据盘，因此目前仅支持指定一个数据盘。
	DataDisk AliyunDisk `toml:"data_disk"`

	// HostName 是该实例的主机名称，可以设置为好记的名称。
	HostName string `toml:"hostname" default:"aliyunmc-server" comment:"实例操作系统的主机名"`

	// RootPassword 是该实例的根用户密码。
	RootPassword string `toml:"root_password,commented" validate:"required" comment:"实例操作系统根用户密码"`

	// ProdPassword 是部署时创建的生产用户 mc 的密码。
	ProdPassword string `toml:"prod_password,commented" validate:"required" comment:"实例操作系统主用户的密码"`

	// SpotInterruptionBehavior 是抢占式实例被中止时的动作，取值为 Stop 或 Terminate。
	SpotInterruptionBehavior string `toml:"spot_interruption_behavior" default:"Stop" validate:"oneof=Stop Terminate" comment:"实例的中止动作，取值Stop或Terminate"`

	// SecurityGroupId 是实例的安全组 ID，需要放行 25565 与 25575 端口。
	SecurityGroupId string `toml:"security_group_id,commented" validate:"required" comment:"实例安全组ID"`
}

// AliyunDisk 对应 [aliyun.ecs.system_disk] 与 [aliyun.ecs.data_disk]，
// 表示云服务器一个云盘的配置项，参考 https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
type AliyunDisk struct {
	// Category 是云盘的种类，通常使用 cloud_essd。
	Category string `toml:"category" default:"cloud_essd" comment:"云盘类型"`

	// Size 是云服务器云盘的大小，单位 GiB
	Size int `toml:"size,commented" validate:"required,min=20" comment:"云盘大小，单位GiB（系统盘常见 40，数据盘常见 100）"`
}

// EcsEndpoint 返回云服务器（ECS）的服务地址，由 RegionId 决定。
func (c Aliyun) EcsEndpoint() string {
	return fmt.Sprintf("ecs.%s.aliyuncs.com", c.RegionId)
}

// OssEndpoint 返回对象存储（OSS）的服务地址，由 RegionId 决定。
func (c Aliyun) OssEndpoint() string {
	return fmt.Sprintf("oss-%s.aliyuncs.com", c.RegionId)
}

// VpcEndpoint 返回专有网络（VPC）的服务地址，由 RegionId 决定。
func (c Aliyun) VpcEndpoint() string {
	return fmt.Sprintf("vpc.%s.aliyuncs.com", c.RegionId)
}
