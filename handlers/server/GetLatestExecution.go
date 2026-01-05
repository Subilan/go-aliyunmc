package server

import (
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleGetLatestSuccessBackup() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		backup, err := store.GetLatestSuccessCommandExecByType(globals.CmdTypeBackupWorlds)

		if err != nil {
			return nil, err
		}

		return helpers.Data(backup), nil
	})
}

func HandleGetLatestSuccessArchive() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		backup, err := store.GetLatestSuccessCommandExecByType(globals.CmdTypeArchiveServer)

		if err != nil {
			return nil, err
		}

		return helpers.Data(backup), nil
	})
}
