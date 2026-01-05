package server

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleGetBackupInfo() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		info, err := store.GetExistingBackups()

		if err != nil {
			return nil, err
		}

		return helpers.Data(info), nil
	})
}
