package helpers

import "github.com/gin-gonic/gin"

func Details(str string) gin.H {
	return gin.H{"Details": str}
}

func Data(d any) gin.H {
	return gin.H{"Data": d}
}

func PaginData(d any, cnt int, nextToken string) gin.H {
	return gin.H{"Data": d, "cnt": cnt, "nextToken": nextToken}
}
