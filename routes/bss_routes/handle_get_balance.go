package bss_routes

import (
	"go-aliyunmc/aliyun"
	"strconv"

	"github.com/gin-gonic/gin"
)

func HandleGetBalance(c *gin.Context) (any, error) {
	response, err := aliyun.BssClient.QueryAccountBalance()
	if err != nil {
		return nil, err
	}

	parsed, err := strconv.ParseFloat(*response.Body.Data.AvailableAmount, 64)

	if err != nil {
		return nil, err
	}

	return parsed, nil
}
