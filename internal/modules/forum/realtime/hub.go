package realtime

import (
	"sync"
	"sync/atomic"

	"github.com/Subilan/go-aliyunmc/internal/common/sse"
)

type subscription struct {
	userID     uint
	ch         chan sse.Event
	onOverflow func()
}

// Hub 同时支持按用户定向投递与全局广播，每个订阅只使用一个 channel。
type Hub struct {
	mu     sync.Mutex
	byUser map[uint]map[*subscription]struct{}
	all    map[*subscription]struct{}
	closed bool
}

// NewHub 创建一个未关闭的 Hub。
func NewHub() *Hub {
	return &Hub{
		byUser: make(map[uint]map[*subscription]struct{}),
		all:    make(map[*subscription]struct{}),
	}
}

// Subscribe 注册一个订阅。initial 会在注册前写入该订阅自己的 channel，
// 用于只向新连接发送 ready 快照。buffer 小于 initial 数量时会自动放大。
func (h *Hub) Subscribe(userID uint, buffer int, onOverflow func(), initial ...sse.Event) (<-chan sse.Event, func(), bool) {
	if buffer < 0 {
		buffer = 0
	}
	if buffer < len(initial) {
		buffer = len(initial)
	}

	ch := make(chan sse.Event, buffer)
	for _, event := range initial {
		ch <- event
	}

	s := &subscription{userID: userID, ch: ch, onOverflow: onOverflow}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		close(ch)
		return ch, func() {}, false
	}
	if h.byUser[userID] == nil {
		h.byUser[userID] = make(map[*subscription]struct{})
	}
	h.byUser[userID][s] = struct{}{}
	h.all[s] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if _, ok := h.all[s]; ok {
				h.removeLocked(s)
			}
			h.mu.Unlock()
		})
	}

	return ch, unsubscribe, true
}

// PublishUser 向指定用户的所有连接投递事件。
func (h *Hub) PublishUser(userID uint, event sse.Event) {
	var overflowed []func()

	h.mu.Lock()
	for s := range h.byUser[userID] {
		if !h.sendLocked(s, event) {
			h.removeLocked(s)
			overflowed = append(overflowed, s.onOverflow)
		}
	}
	h.mu.Unlock()

	invokeAll(overflowed)
}

// PublishAll 向所有连接广播事件。
func (h *Hub) PublishAll(event sse.Event) {
	var overflowed []func()

	h.mu.Lock()
	for s := range h.all {
		if !h.sendLocked(s, event) {
			h.removeLocked(s)
			overflowed = append(overflowed, s.onOverflow)
		}
	}
	h.mu.Unlock()

	invokeAll(overflowed)
}

// HasSubscriber 表示该用户是否至少有一条活跃连接。
func (h *Hub) HasSubscriber(userID uint) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byUser[userID]) > 0
}

// Len 返回当前订阅总数。
func (h *Hub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.all)
}

// Close 关闭所有订阅并拒绝后续订阅。可重复调用。
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}
	h.closed = true

	for s := range h.all {
		delete(h.all, s)
		delete(h.byUser[s.userID], s)
		close(s.ch)
	}
	h.byUser = make(map[uint]map[*subscription]struct{})
}

// sendLocked 非阻塞发送；返回 false 表示缓冲区已满。
func (h *Hub) sendLocked(s *subscription, event sse.Event) bool {
	select {
	case s.ch <- event:
		return true
	default:
		return false
	}
}

// removeLocked 从两个索引中移除订阅并关闭 channel。
func (h *Hub) removeLocked(s *subscription) {
	delete(h.all, s)
	if subs := h.byUser[s.userID]; subs != nil {
		delete(subs, s)
		if len(subs) == 0 {
			delete(h.byUser, s.userID)
		}
	}
	close(s.ch)
}

func invokeAll(callbacks []func()) {
	for _, fn := range callbacks {
		if fn != nil {
			fn()
		}
	}
}

var current atomic.Pointer[Hub]

// Initialize 创建并安装进程级 Hub，返回新实例。
func Initialize() *Hub {
	h := NewHub()
	old := current.Swap(h)
	if old != nil {
		old.Close()
	}
	return h
}

// Current 返回当前进程级 Hub；未初始化时为 nil。
func Current() *Hub {
	return current.Load()
}

// PublishUser 是包级便捷方法，Hub 未初始化时安全 no-op。
func PublishUser(userID uint, event sse.Event) {
	if h := current.Load(); h != nil {
		h.PublishUser(userID, event)
	}
}

// PublishAll 是包级便捷方法，Hub 未初始化时安全 no-op。
func PublishAll(event sse.Event) {
	if h := current.Load(); h != nil {
		h.PublishAll(event)
	}
}

// HasSubscriber 是包级便捷方法，Hub 未初始化时返回 false。
func HasSubscriber(userID uint) bool {
	h := current.Load()
	return h != nil && h.HasSubscriber(userID)
}
