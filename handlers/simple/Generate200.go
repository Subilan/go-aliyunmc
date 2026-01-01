package simple

import "github.com/gin-gonic/gin"

func HandleGenerate200() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{})
	}
}
