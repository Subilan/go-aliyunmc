package config

type BaseConfig struct {
	Expose    int    `toml:"expose" validate:"required"`
	JwtSecret string `toml:"jwt_secret" validate:"required"`
}
