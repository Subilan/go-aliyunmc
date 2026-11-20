// Package messages 从 $GO_ALIYUNMC_STATE_DIR/config/messages.toml 读取可编辑的消息模板。
// 模板使用 Go text/template 语法，每次渲染都会重新读取文件，修改后无需重启。
package messages

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pelletier/go-toml/v2"

	"github.com/Subilan/go-aliyunmc/internal/common/botapi"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/paths"
)

// FileName 是消息模板文件名，位于配置目录下。
const FileName = "messages.toml"

// ServerStarted 是 oneclick_start_game 成功完成后的群消息模板键。
const ServerStarted = "server_started"

// ServerReleased 是实例被删除或释放后的群消息模板键。
const ServerReleased = "server_released"

// IdleCountdownReminder 是空服倒计时临近释放时的群消息模板键。
const IdleCountdownReminder = "idle_countdown_reminder"

// PlayerJoined 是玩家加入服务器时的群消息模板键。
const PlayerJoined = "player_joined"

// PlayerLeft 是玩家离开服务器时的群消息模板键。
const PlayerLeft = "player_left"

// Send 渲染 name 模板并异步发送到 QQ 群。
// 文件缺失、模板渲染失败或发送失败都只记录日志，不影响调用方。
func Send(name string, data any) {
	text, err := Render(name, data)
	if err != nil {
		log_util.Warn("准备群消息失败: template=%s error=%v", name, err)
		return
	}

	botapi.NotifyGroup(text)
}

// Render 读取消息模板文件并渲染名为 name 的模板。
// 模板可以引用 data 的字段，例如 {{.Address}}。
func Render(name string, data any) (string, error) {
	path := filepath.Join(paths.ConfigDir(), FileName)

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取消息模板文件失败(%s): %w", path, err)
	}

	var entries map[string]string
	if err := toml.Unmarshal(raw, &entries); err != nil {
		return "", fmt.Errorf("解析消息模板文件失败(%s): %w", path, err)
	}

	text, ok := entries[name]
	if !ok {
		return "", fmt.Errorf("消息模板 %q 未定义(%s)", name, path)
	}

	tpl, err := template.New(name).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("解析消息模板 %q 失败: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染消息模板 %q 失败: %w", name, err)
	}

	rendered := strings.TrimSpace(buf.String())
	if rendered == "" {
		return "", fmt.Errorf("消息模板 %q 渲染结果为空", name)
	}

	return rendered, nil
}
