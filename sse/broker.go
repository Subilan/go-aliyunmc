package sse

// Broker SSE代理，管理多个客户端连接
type Broker struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	stop chan struct{}
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
	b.stop <- struct{}{}
	close(b.stop)
	close(b.broadcast)
	close(b.unregister)
	close(b.register)
}

// Run 运行广播通道
func (b *Broker) Run() {
	for {
		select {
		case <-b.stop:
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
		}
	}
}

// Register 注册客户端
func (b *Broker) Register(client *Client) {
	b.register <- client
}

// Unregister 注销客户端
func (b *Broker) Unregister(client *Client) {
	b.unregister <- client
}

// Broadcast 广播事件到所有客户端
func (b *Broker) Broadcast(event Event) {
	b.broadcast <- event
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
