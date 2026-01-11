package helpers

import (
	"github.com/gin-gonic/gin"
)

func Details(str string) gin.H {
	return gin.H{"details": str}
}

func Data(d any) gin.H {
	return gin.H{"data": d}
}
