package config

// SSH 对应 [ssh]：控制器登录实例时使用的凭据。
type SSH struct {
	// PrivateKeyPath 是控制器以 mc 用户登录实例所用的私钥文件路径。
	PrivateKeyPath string `toml:"private_key_path,commented" validate:"required" comment:"控制器登录实例所用的私钥文件路径，必须是无口令私钥"`
}
