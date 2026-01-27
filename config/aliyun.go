package config

import "fmt"

// AliyunConfig 包括与阿里云相关的所有配置项目。
type AliyunConfig struct {
	// RegionId 表示系统运作时所考虑的实例所在的地域。关于地域和可用区，详细请参考阿里云官方文档 https://help.aliyun.com/document_detail/40654.html
	//
	// 关于地域：地域决定了实例与 OSS 之间的内网连通性。如果实例与对象存储（OSS）的存储桶（bucket）不处于同一个地域，则它们无法进行内网传输。
	//
	// 内网传输的速度一般可以达到 100MB/s 以上，且流量免费。因此，内网传输是本系统能够高效运行的一个关键因素。内网传输使得一定大小范围内的服务器归档能在可接受的时间范围内在实例与 OSS 之间进行传输（通常在 1～2 分钟）。
	//
	// 公网传输并不能达到这一点，因此本系统默认且仅支持使用内网传输。
	//
	// 也正因此，地域自一开始就需要确定。若地域发生了更换，则需要重新创建相应地域的 OSS bucket 并进行数据的迁移，迁移的过程需要使用公网，此过程会产生流量费用，且速度取决于家用带宽和对象存储带宽。
	RegionId string `toml:"region_id" validate:"required" comment:"实例运行的地域。地域将影响实例与OSS之间的内网连通性，不同地域之间抢占式实例的库存情况也有所不同。"`

	// AccessKeyId 是系统访问阿里云服务的 AKID
	AccessKeyId string `toml:"access_key_id" validate:"required" comment:"用于访问阿里云服务的AKID，可在阿里云RAM控制台中生成"`

	// AccessKeySecret 是系统访问阿里云服务器的 AK 密钥
	AccessKeySecret string `toml:"access_key_secret" validate:"required" comment:"用于访问阿里云服务的AKSecret"`

	// Ecs 包含了对云服务器 ECS 服务的相关配置
	Ecs AliyunEcsConfig `toml:"ecs" validate:"required"`
}

// AliyunEcsConfig 包含了所有与云服务器 ECS 相关的配置内容。参考 https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
type AliyunEcsConfig struct {
	// InternetMaxBandwidthOut 是实例的峰值带宽，单位 Mbps，取值范围 1～100 且为整数
	InternetMaxBandwidthOut int `toml:"internet_max_bandwidth_out" validate:"required,min=1,max=100" comment:"实例的峰值带宽，单位为Mbps，取值范围1～100，必须为整数"`

	// ImageId 是实例使用的系统镜像名称，可以认为决定了操作系统的类型。
	// 取值可以参考 https://help.aliyun.com/zh/ecs/user-guide/find-an-image https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
	ImageId string `toml:"image_id" validate:"required" comment:"实例使用的系统镜像名，决定操作系统类型"`

	// SystemDisk 是该实例使用的系统盘配置。
	SystemDisk EcsDiskConfig `toml:"system_disk" validate:"required"`

	// DataDisk 是该实例使用的第一个数据盘配置。一般不认为需要使用第二个数据盘，因此目前仅支持指定一个数据盘。
	DataDisk EcsDiskConfig `toml:"data_disk" validate:"required"`

	// HostName 是该实例的主机名称，可以设置为好记的名称。
	HostName string `toml:"hostname" validate:"required" comment:"实例操作系统的主机名"`

	// RootPassword 是该实例的根用户密码。
	RootPassword string `toml:"root_password" validate:"required" comment:"实例操作系统根用户密码"`

	// ProdPassword 是该实例的生产用户密码。根用户由于安全性相关考虑，只适合在系统部署时使用，之后都不应使用。
	//
	// 生产用户是指用于实际操作的用户。该用户在系统部署时被创建，用户名为“mc”
	ProdPassword string `toml:"prod_password" validate:"required" comment:"实例操作系统主用户的密码"`

	// SpotInterruptionBehavior 决定抢占式实例如果被中止时的动作，取值为 Stop 或 Terminate
	//  - Stop 将使实例进入节省停机模式，计算资源（CPU、内存）会被释放，而云盘不会且会继续计费。使用这种模式可以避免极端情况（库存短缺）的数据丢失，推荐使用。
	//  - Terminate 将使实例直接被释放，所有数据均被删除。使用此选项时，请注意设置合理的自动备份和归档间隔，否则将导致大幅度回档。
	SpotInterruptionBehavior string `toml:"spot_interruption_behavior" validate:"required,oneof=Stop Terminate" comment:"实例的中止动作，取值Stop或Terminate"`

	// SecurityGroupId 是该实例的安全组 ID。安全组是阿里云提供的一种功能，其中值得注意的是其作为防火墙的功能。
	//
	// 后端能够 Ping 和查询服务器的前提是安全组内的策略允许服务所在的 IP 访问其 25565 和 25575 端口。除此之外其它的端口允许需求也需要在安全组内设置。
	//
	// 安全组相关文档：https://help.aliyun.com/zh/ecs/user-guide/overview-44?
	SecurityGroupId string `toml:"security_group_id" validate:"required" comment:"实例安全组ID"`
}

// EcsEndpoint 返回云服务器（ECS）的服务地址，由 RegionId 决定。
func (c AliyunConfig) EcsEndpoint() string {
	return fmt.Sprintf("ecs.%s.aliyuncs.com", c.RegionId)
}

// OssEndpoint 返回对象存储（OSS）的服务地址，由 RegionId 决定。
func (c AliyunConfig) OssEndpoint() string {
	return fmt.Sprintf("oss-%s.aliyuncs.com", c.RegionId)
}

// VpcEndpoint 返回专有网络（VPC）的服务地址，由 RegionId 决定。
func (c AliyunConfig) VpcEndpoint() string {
	return fmt.Sprintf("vpc.%s.aliyuncs.com", c.RegionId)
}

// EcsDiskConfig 表示云服务器一个云盘的配置项，参考 https://api.aliyun.com/api/Ecs/2014-05-26/CreateInstance
type EcsDiskConfig struct {
	// Category 是云服务器云盘的种类，如无特殊情况建议使用 cloud_essd，其余值有可能导致创建失败
	//
	// 可能的取值：cloud_efficiency cloud_ssd cloud_essd cloud cloud_auto cloud_essd_entry
	Category string `toml:"category" validate:"required" comment:"云盘类型"`

	// Size 是云服务器云盘的大小，单位 GiB
	Size int `toml:"size" validate:"required,min=20" comment:"云盘大小，单位GiB"`
}
