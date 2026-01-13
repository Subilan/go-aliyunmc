package handlers

import (
	"log"
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/stream"
	"github.com/gin-gonic/gin"
	"go.jetify.com/sse"
)

func HandleBeginSimplePublicStream() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		conn, err := sse.Upgrade(ctx, c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, helpers.Details(err.Error()))
			return
		}
		defer conn.Close()

		incomingEvents := stream.SubscribeGlobalStream()
		defer stream.UnsubscribeGlobalStream(incomingEvents)

		for {
			select {
			case e := <-incomingEvents:
				err = conn.SendEvent(ctx, e)

				if err != nil {
					log.Println("cannot send event:", err)
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}
}
