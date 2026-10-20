package config

// UserRoutes 对应 [user_routes]。
type UserRoutes struct {
	ChatToken UserRoutesChatToken `toml:"chat_token"`
}

// UserRoutesChatToken 对应 [user_routes.chat_token]。
type UserRoutesChatToken struct {
	Secret        string `toml:"secret,commented" validate:"required" comment:"JWT签名密钥"`
	ExpireSeconds int    `toml:"expire_seconds" default:"3600" comment:"Token有效期（秒）"`
}
