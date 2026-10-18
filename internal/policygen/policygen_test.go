package policygen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/testutil"
)

func TestMain(m *testing.M) {
	if err := testutil.ChdirProjectRoot(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestRenderExample(t *testing.T) {
	cfg, err := Load(ExampleConfigPath)
	if err != nil {
		t.Fatalf("加载示例配置失败: %v", err)
	}

	out, err := Render(cfg, DefaultTemplatePath, "")
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	for _, want := range []string{
		"# environment: development",
		"# policy: aliyunmc-dev-policy",
		"# environment: production",
		"# policy: aliyunmc-prod-policy",
		"0000000000000000",
		"your-dev-bucket",
		"your-prod-bucket",
		"bss:DescribeAcccount",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("输出缺少 %q", want)
		}
	}

	for _, unwanted := range []string{"{{", "bss:QueryAccountBalance", "acs:RequestedRegion"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("输出不应包含 %q", unwanted)
		}
	}
}

func TestRenderIsolatesBuckets(t *testing.T) {
	cfg, err := Load(ExampleConfigPath)
	if err != nil {
		t.Fatalf("加载示例配置失败: %v", err)
	}

	prod, err := Render(cfg, DefaultTemplatePath, "production")
	if err != nil {
		t.Fatalf("渲染生产策略失败: %v", err)
	}
	if !json.Valid([]byte(prod)) {
		t.Errorf("-env 输出应当是纯 JSON: %s", prod)
	}
	if !strings.Contains(prod, "your-prod-bucket") {
		t.Error("生产策略缺少生产 bucket")
	}
	if strings.Contains(prod, "your-dev-bucket") {
		t.Error("生产策略不应包含开发 bucket")
	}
	if !strings.Contains(prod, "0000000000000000") {
		t.Error("生产策略缺少账号 ID")
	}
	if !strings.Contains(prod, "your-prod-bucket/*") {
		t.Error("空 oss_prefix 应授权整个 bucket")
	}

	dev, err := Render(cfg, DefaultTemplatePath, "development")
	if err != nil {
		t.Fatalf("渲染开发策略失败: %v", err)
	}
	if !strings.Contains(dev, "your-dev-bucket") {
		t.Error("开发策略缺少开发 bucket")
	}
	if strings.Contains(dev, "your-prod-bucket") {
		t.Error("开发策略不应包含生产 bucket")
	}
	if !strings.Contains(dev, "0000000000000000") {
		t.Error("开发策略缺少账号 ID")
	}
}

func TestRenderNormalizesPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")
	content := `
[common]
account_id = "1234567890123456"

[environments.production]
name = "prod-policy"
region_id = "cn-hangzhou"
oss_bucket = "example-bucket"
oss_prefix = "dev"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("加载临时配置失败: %v", err)
	}

	out, err := Render(cfg, DefaultTemplatePath, "production")
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if !strings.Contains(out, "example-bucket/dev/*") {
		t.Errorf("oss_prefix 未规范化为 dev/: %s", out)
	}
}

func TestRenderUnknownEnvironment(t *testing.T) {
	cfg, err := Load(ExampleConfigPath)
	if err != nil {
		t.Fatalf("加载示例配置失败: %v", err)
	}

	if _, err := Render(cfg, DefaultTemplatePath, "staging"); err == nil {
		t.Fatal("未知环境应当报错")
	}
}

func TestLoadRejectsUnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")
	content := `
[common]
account_id = "1234567890123456"

[environments.production]
name = "prod-policy"
region_id = "cn-hangzhou"
oss_bucket = "example-bucket"
unknown_key = "x"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("未知键应当报错")
	}
}

func TestLoadRejectsMissingFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")
	content := `
[common]
account_id = "1234567890123456"

[environments.production]
name = "prod-policy"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("缺少 region_id / oss_bucket 应当报错")
	}
}
