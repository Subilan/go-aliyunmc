package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/store"
	"github.com/Subilan/gomc-server/remote"
	"github.com/gin-gonic/gin"
)

func Start() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance present"}
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		output, err := remote.RunCommandsOnHostSync(ctx, activeInstance.Ip, []string{"/home/mc/server/archive/start.sh"})

		if err != nil {
			return helpers.Data(gin.H{"error": err.Error(), "output": string(output)}), nil
		}

		return helpers.Data(gin.H{"error": nil, "output": string(output)}), nil
	})
}
