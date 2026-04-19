package context_util

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func ParamUint(c *gin.Context, key string) (uint, bool) {
	param := c.Param(key)
	if param == "" {
		return 0, false
	}
	result, err := strconv.ParseUint(param, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(result), true
}