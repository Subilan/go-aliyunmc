package botapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestNewUsesLoopbackBaseURL(t *testing.T) {
	client := New(8081)
	if client.baseURL != "http://127.0.0.1:8081" {
		t.Fatalf("基础地址不正确: %s", client.baseURL)
	}
}

func TestSendGroupMessage(t *testing.T) {
	var got struct {
		Message string `json:"message"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("请求方法应为 POST，得到 %s", r.Method)
		}
		if r.URL.Path != groupMessagePath {
			t.Errorf("请求路径不正确: %s", r.URL.Path)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("Content-Type 不正确: %s", contentType)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("读取请求体失败: %v", err)
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("请求体不是合法 JSON: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true,"group_openid":"group-1","message_id":"message-1"}`)
	}))
	defer server.Close()

	client := newClient(server.URL)
	result, err := client.SendGroupMessage(context.Background(), "  服务器将在 10 分钟后重启  ")
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}

	if got.Message != "服务器将在 10 分钟后重启" {
		t.Errorf("消息应去除首尾空白后发送，得到 %q", got.Message)
	}
	if result.GroupOpenID != "group-1" {
		t.Errorf("group_openid 不正确: %s", result.GroupOpenID)
	}
	if result.MessageID == nil || *result.MessageID != "message-1" {
		t.Errorf("message_id 不正确: %v", result.MessageID)
	}
}

func TestSendGroupMessageRejectsBlank(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	client := newClient(server.URL)
	_, err := client.SendGroupMessage(context.Background(), " \n\t ")
	if !errors.Is(err, ErrEmptyMessage) {
		t.Fatalf("空白消息应返回 ErrEmptyMessage，得到 %v", err)
	}
	if requests != 0 {
		t.Fatalf("空白消息不应发起请求，实际发起 %d 次", requests)
	}
}

func TestSendGroupMessageNullMessageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true,"group_openid":"group-1","message_id":null}`)
	}))
	defer server.Close()

	result, err := newClient(server.URL).SendGroupMessage(context.Background(), "hello")
	if err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	if result.MessageID != nil {
		t.Errorf("message_id 为 null 时应返回 nil，得到 %v", *result.MessageID)
	}
}

func TestSendGroupMessageAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"detail":"QQ 机器人尚未连接"}`)
	}))
	defer server.Close()

	_, err := newClient(server.URL).SendGroupMessage(context.Background(), "hello")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError，得到 %v", err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("状态码不正确: %d", apiErr.StatusCode)
	}
	if apiErr.Detail != "QQ 机器人尚未连接" {
		t.Errorf("detail 不正确: %s", apiErr.Detail)
	}
	if !strings.Contains(apiErr.Error(), "QQ 机器人尚未连接") {
		t.Errorf("错误信息应包含 detail: %s", apiErr.Error())
	}
}

func TestSendGroupMessageValidationErrorDetailIsArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"detail":[{"loc":["body","message"],"msg":"field required","type":"missing"}]}`)
	}))
	defer server.Close()

	_, err := newClient(server.URL).SendGroupMessage(context.Background(), "hello")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应返回 *APIError，得到 %v", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("状态码不正确: %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Detail, "field required") {
		t.Errorf("detail 应包含数组中的原因: %s", apiErr.Detail)
	}
}

func TestSendGroupMessageInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `not-json`)
	}))
	defer server.Close()

	_, err := newClient(server.URL).SendGroupMessage(context.Background(), "hello")
	if err == nil || !strings.Contains(err.Error(), "解析") {
		t.Fatalf("非法响应应返回解析错误，得到 %v", err)
	}
}

func TestSendGroupMessageTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := server.URL
	server.Close()

	_, err := newClient(url).SendGroupMessage(context.Background(), "hello")
	if err == nil {
		t.Fatal("机器人不可达时应返回错误")
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("连接失败不应是 *APIError，得到 %v", err)
	}
	if !strings.Contains(err.Error(), "调用机器人接口失败") {
		t.Fatalf("错误信息不正确: %v", err)
	}
}

func TestParseDetailFallsBackToRawBody(t *testing.T) {
	if got := parseDetail([]byte("  plain text  ")); got != "plain text" {
		t.Errorf("非 JSON 响应应回退为原始文本，得到 %q", got)
	}
}

func TestTruncateDetail(t *testing.T) {
	got := truncateDetail(strings.Repeat("测", maxDetailBytes))
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("超长 detail 应带省略号: %q", got)
	}
	if len(got) > maxDetailBytes+len("…") {
		t.Fatalf("截断结果过长: %d 字节", len(got))
	}
	if !utf8.ValidString(got) {
		t.Fatal("截断不应破坏 UTF-8")
	}
	if short := truncateDetail("  short  "); short != "short" {
		t.Fatalf("短文本应只去除空白: %q", short)
	}
}

func TestSendGroupMessageAsync(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body groupMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		received <- body.Message

		_, _ = io.WriteString(w, `{"ok":true,"group_openid":"group-1","message_id":"message-1"}`)
	}))
	defer server.Close()

	newClient(server.URL).SendGroupMessageAsync("  服务器将在 10 分钟后重启  ")

	select {
	case message := <-received:
		if message != "服务器将在 10 分钟后重启" {
			t.Errorf("消息不正确: %q", message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("后台消息没有到达机器人")
	}
}

func TestSendGroupMessageAsyncReturnsBeforeResponse(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"detail":"QQ 机器人尚未连接"}`)
	}))
	defer server.Close()

	returned := make(chan struct{})
	go func() {
		newClient(server.URL).SendGroupMessageAsync("hello")
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("异步发送不应等待机器人响应")
	}

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("后台发送没有发出请求")
	}
	close(release)
}

func TestSendGroupMessageAsyncIgnoresBlankMessage(t *testing.T) {
	requested := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requested <- struct{}{}
	}))
	defer server.Close()

	newClient(server.URL).SendGroupMessageAsync(" \n\t ")

	select {
	case <-requested:
		t.Fatal("空白消息不应发起请求")
	case <-time.After(100 * time.Millisecond):
	}
}
