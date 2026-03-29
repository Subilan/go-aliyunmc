package config

type AutotlsConfig struct {
	Enabled bool     `toml:"enabled" comment:"是否启用"`
	Domains []string `toml:"domains" comment:"签发域名" validate:"omitempty,dive,min=1"`
}