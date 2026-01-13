package stream

import (
	"log"

	"github.com/Subilan/go-aliyunmc/broker"
	"go.jetify.com/sse"
)

var publicChannel = broker.New[*sse.Event]()
var publicChannelInitialized = false

func InitPublicChannel() {
	if publicChannelInitialized {
		log.Fatalln("reinitializing global stream is not permitted.")
	}
	go publicChannel.Start()
	publicChannelInitialized = true
}

func SubPublicChannel() chan *sse.Event {
	return publicChannel.Subscribe()
}

func UnsubPublicChannel(ch chan *sse.Event) {
	publicChannel.Unsubscribe(ch)
}
