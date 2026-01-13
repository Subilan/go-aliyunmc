package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
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

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

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

			case <-ticker.C:
				if err := conn.SendComment(ctx, " "); err != nil {
					log.Println("cannot ping client:", err)
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}
}
