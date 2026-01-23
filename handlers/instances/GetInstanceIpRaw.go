package instances

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleGetInstanceIpRaw() gin.HandlerFunc {
	return func(c *gin.Context) {
		fallback := c.Query("fallback")

		activeInstance, err := store.GetIpAllocatedActiveInstance()

		if err != nil {
			c.Data(http.StatusOK, "text/plain", []byte(fallback))
			return
		}

		c.Data(http.StatusOK, "text/plain", []byte(*activeInstance.Ip))
		return
	}
}
