package simple

import "github.com/gin-gonic/gin"

// HandleGenerate200 返回一个 200 状态
func HandleGenerate200() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{})
	}
}
