package user_routes

import "github.com/Subilan/go-aliyunmc/utils"

var C Config

type Config struct {
	ChatToken ChatTokenConfig `toml:"chat_token" validate:"required"`
	BotToken  BotTokenConfig  `toml:"bot_token" validate:"required"`
}

type ChatTokenConfig struct {
	Secret        string `toml:"secret" validate:"required" comment:"JWT签名密钥"`
	ExpireSeconds int    `toml:"expire_seconds" validate:"required" comment:"Token有效期（秒）"`
}

type BotTokenConfig struct {
	Secret        string `toml:"secret" validate:"required" comment:"BotToken JWT签名密钥"`
	ExpireSeconds int    `toml:"expire_seconds" validate:"required" comment:"BotToken有效期（秒），一般设置为3个月=7776000"`
}

func init() {
	utils.MustBindConfigToml(&C, "user-routes")
}
