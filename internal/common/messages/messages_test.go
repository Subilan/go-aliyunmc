package messages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Subilan/go-aliyunmc/internal/common/paths"
)

// writeMessages 在临时状态目录下写入 config/messages.toml，并把它设为当前配置目录。
func writeMessages(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv(paths.StateDirEnv, dir)

	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}

	path := filepath.Join(configDir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入消息模板失败: %v", err)
	}
	return path
}

func TestRender(t *testing.T) {
	writeMessages(t, `server_started = "服务器已开启~"`)

	got, err := Render(ServerStarted, nil)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if got != "服务器已开启~" {
		t.Fatalf("渲染结果不正确: %q", got)
	}
}

func TestRenderWithData(t *testing.T) {
	writeMessages(t, `server_started = "  服务器已开启~ 地址：{{.Address}}  "`)

	got, err := Render(ServerStarted, map[string]any{"Address": "203.0.113.10:25565"})
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}
	if got != "服务器已开启~ 地址：203.0.113.10:25565" {
		t.Fatalf("渲染结果不正确: %q", got)
	}
}

func TestRenderReloadsFile(t *testing.T) {
	path := writeMessages(t, `server_started = "first"`)

	if got, err := Render(ServerStarted, nil); err != nil || got != "first" {
		t.Fatalf("首次渲染不正确: %q, %v", got, err)
	}

	if err := os.WriteFile(path, []byte(`server_started = "second"`), 0o644); err != nil {
		t.Fatalf("修改消息模板失败: %v", err)
	}

	if got, err := Render(ServerStarted, nil); err != nil || got != "second" {
		t.Fatalf("修改后应重新读取文件，得到 %q, %v", got, err)
	}
}

func TestRenderMissingFile(t *testing.T) {
	t.Setenv(paths.StateDirEnv, t.TempDir())

	if _, err := Render(ServerStarted, nil); err == nil || !strings.Contains(err.Error(), "读取消息模板文件失败") {
		t.Fatalf("文件不存在时应返回读取错误，得到 %v", err)
	}
}

func TestRenderMissingKey(t *testing.T) {
	writeMessages(t, `other = "x"`)

	if _, err := Render(ServerStarted, nil); err == nil || !strings.Contains(err.Error(), "未定义") {
		t.Fatalf("缺少模板键时应返回错误，得到 %v", err)
	}
}

func TestRenderInvalidTOML(t *testing.T) {
	writeMessages(t, `not toml`)

	if _, err := Render(ServerStarted, nil); err == nil || !strings.Contains(err.Error(), "解析消息模板文件失败") {
		t.Fatalf("非法 TOML 应返回解析错误，得到 %v", err)
	}
}

func TestRenderTemplateError(t *testing.T) {
	writeMessages(t, `server_started = "{{.Missing}"`)

	if _, err := Render(ServerStarted, nil); err == nil || !strings.Contains(err.Error(), "解析消息模板") {
		t.Fatalf("模板语法错误应返回错误，得到 %v", err)
	}
}

func TestRenderMissingDataKey(t *testing.T) {
	writeMessages(t, `server_started = "{{.Address}}"`)

	if _, err := Render(ServerStarted, map[string]any{}); err == nil || !strings.Contains(err.Error(), "渲染消息模板") {
		t.Fatalf("模板引用缺失字段时应返回错误，得到 %v", err)
	}
}

func TestRenderBlankResult(t *testing.T) {
	writeMessages(t, `server_started = "   "`)

	if _, err := Render(ServerStarted, nil); err == nil || !strings.Contains(err.Error(), "为空") {
		t.Fatalf("空白模板应返回错误，得到 %v", err)
	}
}
