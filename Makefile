# 生成配置文件
gen-config:
	@go run ./cmd/configgen

# 运行测试
test:
	@mkdir -p configs
	@cp -n example_configs/*.toml configs/ 2>/dev/null || true
	@go test -v ./...

# 构建项目
build:
	@go build -o aliyunmc .

# 运行项目
dev:
	@GO_ALIYUNMC_DEV=1 go run . run

run:
	@go run . run