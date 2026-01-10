package config

type MonitorConfig struct {
	ActiveInstance ActiveInstance `toml:"active_instance" validate:"required"`
	PublicIp       PublicIp       `toml:"public_ip" validate:"required"`
	InstanceCharge InstanceCharge `toml:"instance_charge" validate:"required"`
	Backup         Backup         `toml:"backup" validate:"required"`
	EmptyServer    EmptyServer    `toml:"empty_server" validate:"required"`
	ServerStatus   ServerStatus   `toml:"server_status" validate:"required"`
	StartInstance  StartInstance  `toml:"start_instance" validate:"required"`
}
