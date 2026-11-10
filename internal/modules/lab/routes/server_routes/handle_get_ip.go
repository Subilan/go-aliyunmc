package server_routes

import (
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/dao"

	"github.com/gin-gonic/gin"
)

func HandleGetIP(c *gin.Context) {
	ip, err := dao.GetActiveInstanceIpNonEmpty()

	if err != nil {
		c.String(404, "")
		return
	}

	c.String(200, ip)
}
