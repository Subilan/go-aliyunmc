.PHONY: gen-config gen-policy test build dev run migrate-diff

MIGRATION_NAME ?= change

APP_VERSION := $(shell cat VERSION)
APP_COMMIT := $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)
APP_BUILD_TIME := $(shell date -u +%FT%TZ)
LDFLAGS := -X github.com/Subilan/go-aliyunmc/internal/version.Version=$(APP_VERSION) \
	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=$(APP_COMMIT) \
	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=$(APP_BUILD_TIME)

# 生成配置文件
gen-config:
	@go run ./cmd/configgen

# 根据 policies/policy.toml 渲染 RAM 策略并输出到 stdout
gen-policy:
	@go run . gen_policy

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

# 根据 GORM 模型与已有迁移的差量生成 MySQL 迁移（需要 atlas CLI 和 Docker）
migrate-diff:
	@atlas migrate diff --env mysql $(MIGRATION_NAME)
