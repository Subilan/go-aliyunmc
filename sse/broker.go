package sse

import "sync"

// Broker SSE代理，管理多个客户端连接
type Broker struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	stop       chan struct{}
	stopOnce   sync.Once
}

// NewBroker 创建新的SSE代理
func NewBroker() *Broker {
	return &Broker{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 100),
		stop:       make(chan struct{}),
	}
}

// Stop 停止该广播通道的运行，且不可恢复
func (b *Broker) Stop() {
	b.stopOnce.Do(func() {
		close(b.stop)
	})
}

// Run 运行广播通道
func (b *Broker) Run() {
	for {
		select {
		case <-b.stop:
			for client := range b.clients {
				delete(b.clients, client)
				client.Close()
			}
			return
		case client := <-b.register:
			b.clients[client] = true
		case client := <-b.unregister:
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				client.Close()
			}
		case event := <-b.broadcast:
			for client := range b.clients {
				select {
				case client.eventChan <- event:
				default:
					delete(b.clients, client)
					client.Close()
				}
			}

			// 任务结束事件发出后，服务端结束当前 broker。
			// 注意，此处不应该调用 client.Close()
			if event.Event == "task_done" {
				b.Stop()
				return
			}
		}
	}
}

// Register 注册客户端
func (b *Broker) Register(client *Client) {
	select {
	case <-b.stop:
		client.Close()
	case b.register <- client:
	}
}

// Unregister 注销客户端
func (b *Broker) Unregister(client *Client) {
	select {
	case <-b.stop:
		client.Close()
	case b.unregister <- client:
	}
}

// Broadcast 广播事件到所有客户端
func (b *Broker) Broadcast(event Event) {
	select {
	case <-b.stop:
		return
	case b.broadcast <- event:
	}
}

// BroadcastData 广播数据事件到所有客户端
func (b *Broker) BroadcastData(data interface{}) {
	b.Broadcast(Event{Data: data})
}

// BroadcastEvent 广播命名事件到所有客户端
func (b *Broker) BroadcastEvent(name string, data interface{}) {
	b.Broadcast(Event{Event: name, Data: data})
}

// ClientCount 返回当前客户端数量
func (b *Broker) ClientCount() int {
	return len(b.clients)
}
