package user_routes

import "github.com/Subilan/go-aliyunmc/utils"

var C Config

func MustLoadConfig() {
	utils.MustBindConfigToml(&C, "user-routes")
}

type Config struct {
	ChatToken ChatTokenConfig `toml:"chat_token" validate:"required"`
}

type ChatTokenConfig struct {
	Secret        string `toml:"secret" validate:"required" comment:"JWT签名密钥"`
	ExpireSeconds int    `toml:"expire_seconds" validate:"required" comment:"Token有效期（秒）"`
}
