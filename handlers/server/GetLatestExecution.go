package server

import (
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleGetLatestSuccessBackup() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		backup, err := store.GetLatestSuccessCommandExecByType(consts.CmdTypeBackupWorlds)

		if err != nil {
			return nil, err
		}

		return helpers.Data(backup), nil
	})
}

func HandleGetLatestSuccessArchive() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		backup, err := store.GetLatestSuccessCommandExecByType(consts.CmdTypeArchiveServer)

		if err != nil {
			return nil, err
		}

		return helpers.Data(backup), nil
	})
}
