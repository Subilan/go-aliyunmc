package server_routes

import (
	"github.com/Subilan/go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

func HandleGetIP(c *gin.Context) {
	ip, err := store.GetActiveInstanceIpNonEmpty()

	if err != nil {
		c.String(404, "")
		return
	}

	c.String(200, ip)
}
