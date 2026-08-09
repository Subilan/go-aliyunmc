.PHONY: gen-config test build dev run

APP_VERSION := $(shell cat VERSION)
APP_COMMIT := $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)
APP_BUILD_TIME := $(shell date -u +%FT%TZ)
LDFLAGS := -X github.com/Subilan/go-aliyunmc/internal/version.Version=$(APP_VERSION) \
	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=$(APP_COMMIT) \
	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=$(APP_BUILD_TIME)

# 生成配置文件
gen-config:
	@go run ./cmd/configgen

# 运行测试
test:
	@go test -v ./...

# 构建项目
build:
	@go build -ldflags "$(LDFLAGS)" -o aliyunmc .

# 运行项目
dev:
	@GO_ALIYUNMC_DEV=1 go run . run

run:
	@go run . run
