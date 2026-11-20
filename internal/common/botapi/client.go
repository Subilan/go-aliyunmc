// Package botapi 封装 QQ 群机器人进程在本机回环地址上暴露的 HTTP 接口。
//
// 机器人只接受来自 127.0.0.1 / ::1 的直连请求，并拒绝携带转发头的请求，
// 因此客户端固定连接 127.0.0.1，仅端口由配置提供。
package botapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/log_util"
	"github.com/Subilan/go-aliyunmc/internal/common/str_util"
)

const (
	loopbackHost = "127.0.0.1"

	groupMessagePath = "/bot/group-message"

	requestTimeout   = 10 * time.Second
	dialTimeout      = 3 * time.Second
	maxResponseBytes = 64 << 10
	maxDetailBytes   = 1024
)

// ErrEmptyMessage 表示待发送的消息去除首尾空白后为空。
var ErrEmptyMessage = errors.New("消息内容不能为空")

// Client 是机器人 HTTP 接口的客户端，可被多个 goroutine 并发使用。
type Client struct {
	baseURL string
	http    *http.Client
}

// New 创建指向本机指定端口的客户端。
func New(port int) *Client {
	return newClient("http://" + net.JoinHostPort(loopbackHost, strconv.Itoa(port)))
}

func newClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				// 机器人会拒绝带转发头的请求；显式关闭环境代理，确保直连本机。
				Proxy: nil,
				DialContext: (&net.Dialer{
					Timeout:   dialTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConns:        4,
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

var defaultClient = sync.OnceValue(func() *Client {
	return New(config.Get().Bot.API.Port)
})

// Default 返回按当前配置创建并复用的进程级客户端。
// 必须在配置加载完成后调用；客户端的连接会一直保留到进程退出。
func Default() *Client {
	return defaultClient()
}

// GroupMessageResult 是发送群消息成功后的结果。
type GroupMessageResult struct {
	// GroupOpenID 是实际发送到的群 openid。
	GroupOpenID string `json:"group_openid"`
	// MessageID 是 QQ 返回的消息 id，可能为 null。
	MessageID *string `json:"message_id"`
}

// SendGroupMessage 向机器人配置的固定群发送一条纯文本消息。
// message 会先去除首尾空白；去除后为空时直接返回 ErrEmptyMessage，不发起请求。
func (c *Client) SendGroupMessage(ctx context.Context, message string) (*GroupMessageResult, error) {
	text := strings.TrimSpace(message)
	if text == "" {
		return nil, ErrEmptyMessage
	}

	payload, err := json.Marshal(groupMessageRequest{Message: text})
	if err != nil {
		return nil, fmt.Errorf("编码群消息请求失败: %w", err)
	}

	var response struct {
		GroupMessageResult
		OK bool `json:"ok"`
	}
	if err := c.doJSON(ctx, http.MethodPost, groupMessagePath, payload, &response); err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, errors.New("机器人接口未返回成功标记")
	}

	return &response.GroupMessageResult, nil
}

// SendGroupMessageAsync 在后台发送一条纯文本消息，立即返回，不把失败抛给调用方。
// 发送失败只写一条 warning 日志；消息为空时直接忽略。
// 调用方不应依赖消息送达或发送顺序。
func (c *Client) SendGroupMessageAsync(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}

	go func() {
		// 调用方的 ctx 常随请求或任务结束而取消，后台发送使用独立上下文，
		// 由 http.Client.Timeout 限制单次发送的时长。
		if _, err := c.SendGroupMessage(context.Background(), message); err != nil {
			log_util.Warn("后台发送 QQ 群消息失败: %v", err)
		}
	}()
}

// NotifyGroup 是 SendGroupMessageAsync 的包级快捷方式，使用 Default 客户端。
func NotifyGroup(message string) {
	Default().SendGroupMessageAsync(message)
}

// APIError 表示机器人接口返回了非 200 响应。
type APIError struct {
	// StatusCode 是机器人返回的 HTTP 状态码。
	StatusCode int
	// Detail 是响应体中的 detail 字段；不是标准结构时保存原始响应文本。
	Detail string
}

func (e *APIError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("机器人接口返回 HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("机器人接口返回 HTTP %d：%s", e.StatusCode, e.Detail)
}

type groupMessageRequest struct {
	Message string `json:"message"`
}

// doJSON 发送请求并把 200 响应解码到 out；out 为 nil 时只检查状态码。
func (c *Client) doJSON(ctx context.Context, method, path string, payload []byte, out any) error {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("构造机器人接口请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("调用机器人接口失败: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("读取机器人接口响应失败: %w", err)
	}
	if len(data) > maxResponseBytes {
		return errors.New("机器人接口响应过大")
	}

	if resp.StatusCode != http.StatusOK {
		return &APIError{StatusCode: resp.StatusCode, Detail: parseDetail(data)}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("解析机器人接口响应失败: %w", err)
	}
	return nil
}

// parseDetail 提取错误响应中的 detail。
// FastAPI 的参数校验错误里 detail 是数组而非字符串，这里统一转成可读文本。
func parseDetail(body []byte) string {
	var payload struct {
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.Detail) == 0 {
		return truncateDetail(string(body))
	}

	var detail string
	if err := json.Unmarshal(payload.Detail, &detail); err == nil {
		return truncateDetail(detail)
	}

	return truncateDetail(string(payload.Detail))
}

func truncateDetail(s string) string {
	s = strings.TrimSpace(s)
	truncated := str_util.Truncate(s, maxDetailBytes)
	if len(truncated) == len(s) {
		return s
	}
	return truncated + "…"
}
