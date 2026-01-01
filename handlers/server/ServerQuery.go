package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/helpers"
	"github.com/Subilan/gomc-server/helpers/remote"
	"github.com/Subilan/gomc-server/helpers/store"
	"github.com/gin-gonic/gin"
)

type QueryQuery struct {
	QueryType QueryType `form:"queryType" binding:"required,oneof=sizes screenfetch"`
}

type QueryType string

const (
	QuerySizes       QueryType = "sizes"
	QueryScreenfetch           = "screenfetch"
)

func (q QueryType) Command() []string {
	switch q {
	case QueryScreenfetch:
		return []string{"screenfetch -N"}
	case QuerySizes:
		return []string{"du -sh /home/mc/server/archive", "du -sh /home/mc/server/archive/world", "du -sh /home/mc/server/archive/world_nether", "du -sh /home/mc/server/archive/world_the_end"}
	}

	return nil
}

func HandleServerQuery() gin.HandlerFunc {
	return helpers.QueryHandler[QueryQuery](func(query QueryQuery, c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "no active instance present"}
		}

		ctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()

		cmd := query.QueryType.Command()

		output, err := remote.RunCommandAsProdSync(ctx, activeInstance.Ip, cmd)

		if err != nil {
			return helpers.Data(gin.H{"error": err.Error(), "output": string(output)}), nil
		}

		return helpers.Data(gin.H{"error": nil, "output": string(output)}), nil
	})
}
