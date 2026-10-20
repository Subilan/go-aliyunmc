package version

// 以下变量由构建脚本通过 -ldflags 注入，未注入时保持默认值。
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
