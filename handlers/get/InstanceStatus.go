package get

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
)

type StoredInstanceStatus struct {
	InstanceId string    `json:"instanceId"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func InstanceStatus() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, helpers.HttpError{Code: http.StatusBadRequest, Details: "instanceId not provided"}
		}

		ctx, cancel := context.WithTimeout(c, 1*time.Second)
		defer cancel()

		var result StoredInstanceStatus

		err := globals.Pool.QueryRowContext(ctx, `
SELECT statuses, updated_at FROM instance_statuses WHERE instance_id = $1
`, instanceId).Scan(&result.Status, &result.UpdatedAt)

		if err != nil {
			return nil, err
		}

		result.InstanceId = instanceId

		return helpers.Data(result), nil
	})
}
