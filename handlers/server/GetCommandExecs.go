package server

import (
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type GetCommandExecQuery struct {
	helpers.Paginated
}

func HandleGetCommandExecs() gin.HandlerFunc {
	return helpers.QueryHandler[GetCommandExecQuery](func(query GetCommandExecQuery, c *gin.Context) (any, error) {
		if query.PageSize == 0 {
			query.PageSize = 10
		}

		if query.Page == 0 {
			query.Page = 1
		}

		var results = make([]store.JoinedCommandExec, 0, 10)

		rows, err := db.Pool.Query("SELECT c.id, c.type, c.by, c.created_at, c.updated_at, c.status, c.auto, c.comment, u.username FROM command_exec c LEFT JOIN users u ON c.by=u.id ORDER BY c.created_at DESC LIMIT ? OFFSET ?", query.PageSize, (query.Page-1)*query.PageSize)

		if err != nil {
			return nil, err
		}

		defer rows.Close()
		for rows.Next() {
			var result store.JoinedCommandExec
			err = rows.Scan(&result.Id, &result.Type, &result.By, &result.CreatedAt, &result.UpdatedAt, &result.Status, &result.Auto, &result.Comment, &result.Username)

			if err != nil {
				return nil, err
			}

			results = append(results, result)
		}

		var total int64
		err = db.Pool.QueryRow("SELECT COUNT(*) FROM command_exec").Scan(&total)

		if err != nil {
			return nil, err
		}

		return helpers.Data(gin.H{"total": total, "data": results}), nil
	})
}
