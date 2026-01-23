package config

// MonitorConfig 包含了对系统运行的监控器相关配置
type MonitorConfig struct {
	// ActiveInstance 是对 monitors.ActiveInstance 的相关配置
	ActiveInstance ActiveInstance `toml:"active_instance" validate:"required"`

	// PublicIP 是对 monitors.PublicIP 的相关配置
	PublicIP PublicIP `toml:"public_ip" validate:"required"`

	// InstanceCharge 是对 monitors.InstanceCharge 的相关配置
	InstanceCharge InstanceCharge `toml:"instance_charge" validate:"required"`

	// Backup 是对 monitors.Backup 的相关配置
	Backup Backup `toml:"backup" validate:"required"`

	// EmptyServer 是对 monitors.EmptyServer 的相关配置
	EmptyServer EmptyServer `toml:"empty_server" validate:"required"`

	// ServerStatus 是对 monitors.ServerStatus 的相关配置
	ServerStatus ServerStatus `toml:"server_status" validate:"required"`

	// StartInstance 是对 monitors.StartActiveInstanceWhenReady 的相关配置
	StartInstance StartInstance `toml:"start_instance" validate:"required"`

	// BssSync 是对 monitors.BssSync 的相关配置
	BssSync BssSync `toml:"bss_sync" validate:"required"`

	// Whitelist 是对 monitors.Whitelist 的相关配置
	Whitelist Whitelist `toml:"whitelist" validate:"required"`
}
