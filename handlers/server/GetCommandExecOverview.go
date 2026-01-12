package server

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type CommandExecOverview struct {
	SuccessCount      int                     `json:"successCount"`
	ErrorCount        int                     `json:"errorCount"`
	LatestCommandExec store.JoinedCommandExec `json:"latestCommandExec,omitempty"`
}

func HandleGetCommandExecOverview() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		var result CommandExecOverview

		err := db.Pool.QueryRow("SELECT COUNT(*) FROM command_exec WHERE status='success'").Scan(&result.SuccessCount)
		if err != nil {
			return nil, err
		}

		err = db.Pool.QueryRow("SELECT COUNT(*) FROM command_exec WHERE status='error'").Scan(&result.ErrorCount)

		if err != nil {
			return nil, err
		}

		_ = db.Pool.
			QueryRow("SELECT c.id, c.type, c.by, c.created_at, c.updated_at, c.status, c.auto, c.comment, u.username FROM command_exec c JOIN users u ON c.by=u.id ORDER BY c.created_at DESC LIMIT 1").
			Scan(
				&result.LatestCommandExec.Id,
				&result.LatestCommandExec.Type,
				&result.LatestCommandExec.By,
				&result.LatestCommandExec.CreatedAt,
				&result.LatestCommandExec.UpdatedAt,
				&result.LatestCommandExec.Status,
				&result.LatestCommandExec.Auto,
				&result.LatestCommandExec.Comment,
				&result.LatestCommandExec.Username,
			)

		return helpers.Data(result), nil
	})
}
