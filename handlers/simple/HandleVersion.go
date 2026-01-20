package simple

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/gin-gonic/gin"
)

func HandleVersion() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		return gin.H{"version": "0.0.1"}, nil
	})
}
