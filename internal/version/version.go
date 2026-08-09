package version

// 以下变量由构建脚本通过 -ldflags 注入：
//
//	-X github.com/Subilan/go-aliyunmc/internal/version.Version=1.0.0
//	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=<git sha>
//	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=<RFC3339>
//
// 本地 go run / make dev 未注入时保持 dev 默认值。
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)
