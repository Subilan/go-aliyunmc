package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Event SSE事件
type Event struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// Client SSE客户端
type Client struct {
	writer    gin.ResponseWriter
	flusher   http.Flusher
	ctx       context.Context
	cancel    context.CancelFunc
	eventChan chan Event
}

// NewClient 对当前链接进行升级并创建新的SSE客户端
func NewClient(c *gin.Context) (*Client, error) {
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming unsupported")
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")

	ctx, cancel := context.WithCancel(c.Request.Context())

	return &Client{
		writer:    c.Writer,
		flusher:   flusher,
		ctx:       ctx,
		cancel:    cancel,
		eventChan: make(chan Event, 100),
	}, nil
}

func (c *Client) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Send 发送事件到客户端
func (c *Client) Send(event Event) error {
	select {
	case c.eventChan <- event:
		return nil
	case <-c.ctx.Done():
		return fmt.Errorf("client disconnected")
	}
}

// SendData 发送数据事件
func (c *Client) SendData(data interface{}) error {
	return c.Send(Event{Data: data})
}

// SendEvent 发送命名事件
func (c *Client) SendEvent(name string, data interface{}) error {
	return c.Send(Event{Event: name, Data: data})
}

// Close 结束客户端连接，handler 将返回
func (c *Client) Close() {
	c.cancel()
}

// Listen 监听事件并发送给客户端
func (c *Client) Listen() {
	defer c.Close()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.writeComment("keep-alive")
			c.flusher.Flush()

		case event := <-c.eventChan:
			if err := c.writeEvent(event); err != nil {
				return
			}
			c.flusher.Flush()
			if event.Event == "task_done" {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Client) writeComment(comment string) {
	_, _ = fmt.Fprintf(c.writer, ":%s\n\n", comment)
}

// writeEvent 写入事件到响应流
func (c *Client) writeEvent(event Event) error {
	// 写入事件ID
	if event.ID != "" {
		if _, err := fmt.Fprintf(c.writer, "id: %s\n", event.ID); err != nil {
			return err
		}
	}

	// 写入事件类型
	if event.Event != "" {
		if _, err := fmt.Fprintf(c.writer, "event: %s\n", event.Event); err != nil {
			return err
		}
	}

	// 写入事件数据
	if event.Data != nil {
		data, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.writer, "data: %s\n", data); err != nil {
			return err
		}
	}

	// 写入结束标记
	if _, err := c.writer.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}
