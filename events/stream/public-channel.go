package stream

import (
	"log"

	"github.com/Subilan/go-aliyunmc/broker"
	"go.jetify.com/sse"
)

// publicChannel 本质上是一个 broker.Broker，表示一个面向所有用户的消息频道
var publicChannel = broker.New[*sse.Event]()

// publicChannelInitialized 用于记录 publicChannel 的初始化情况，避免重复初始化
var publicChannelInitialized = false

// InitPublicChannel 开启公共频道
func InitPublicChannel() {
	if publicChannelInitialized {
		log.Fatalln("reinitializing global stream is not permitted.")
	}
	go publicChannel.Start()
	publicChannelInitialized = true
}

// SubPublicChannel 从公共频道上订阅消息，相当于调用 broker.Broker 的 Subscribe
func SubPublicChannel() chan *sse.Event {
	return publicChannel.Subscribe()
}

// UnsubPublicChannel 取消从公共频道上订阅消息，相当于调用 broker.Broker 的 Unsubscribe
func UnsubPublicChannel(ch chan *sse.Event) {
	publicChannel.Unsubscribe(ch)
}
