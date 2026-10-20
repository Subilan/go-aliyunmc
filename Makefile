.PHONY: gen-config gen-policy test build dev run bundle bump bump-push migrate-diff

MIGRATION_NAME ?= change
BUMP ?= patch

APP_VERSION := $(shell cat VERSION)
APP_COMMIT := $(shell git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)
APP_BUILD_TIME := $(shell date -u +%FT%TZ)
LDFLAGS := -X github.com/Subilan/go-aliyunmc/internal/version.Version=$(APP_VERSION) \
	-X github.com/Subilan/go-aliyunmc/internal/version.Commit=$(APP_COMMIT) \
	-X github.com/Subilan/go-aliyunmc/internal/version.BuildTime=$(APP_BUILD_TIME)

# 生成配置文件
gen-config:
	@go run ./cmd/configgen

# 根据 policy/policy.toml 渲染 RAM 策略并输出到 stdout
gen-policy:
	@go run ./cmd/aliyunmc gen_policy

# 运行测试
test:
	@go test -v ./...

# 构建项目
build:
	@go build -ldflags "$(LDFLAGS)" -o aliyunmc ./cmd/aliyunmc

# 生成发布包（linux/amd64），production 仓库的发布流程也使用这个目标
bundle:
	@bash release/bundle.sh "$(OUT)"

# 按 semver 升级版本并打标签，BUMP 为 major|minor|patch 或具体版本号
bump:
	@bash release/version.sh "$(BUMP)"

# 同上，并把提交与标签推送到远端
bump-push:
	@bash release/version.sh "$(BUMP)" --push

# 运行项目
dev:
	@GO_ALIYUNMC_DEV=1 GO_ALIYUNMC_STATE_DIR=.dev go run ./cmd/aliyunmc run

run:
	@GO_ALIYUNMC_STATE_DIR=.dev go run ./cmd/aliyunmc run

# 根据 GORM 模型与已有迁移的差量生成 MySQL 迁移（需要 atlas CLI 和 Docker）
migrate-diff:
	@atlas migrate diff --env mysql $(MIGRATION_NAME)
