package ecsActions

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/gin-gonic/gin"
)

const DeleteInstanceTimeout = 15 * time.Second

func DeleteInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			err := globals.Pool.QueryRow("SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&instanceId)

			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "找不到符合要求的instance id"}
				}

				return nil, err
			}
		}

		ctx, cancel := context.WithTimeout(c, DeleteInstanceTimeout)
		defer cancel()

		var cnt int

		err := globals.Pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM instances WHERE deleted_at IS NULL AND instance_id = ?", instanceId).Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt == 0 {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "instance not found or already deleted"}
		}

		_, force := c.GetQuery("force")
		_, forceStop := c.GetQuery("forceStop")

		deleteInstanceRequest := &ecs20140526.DeleteInstanceRequest{
			InstanceId: &instanceId,
			Force:      &force,
			ForceStop:  &forceStop,
		}

		_, err = globals.EcsClient.DeleteInstance(deleteInstanceRequest)

		if err != nil {
			return nil, err
		}

		_, err = globals.Pool.ExecContext(ctx, "UPDATE instances SET deleted_at = ? WHERE instance_id = ?", time.Now().Local(), instanceId)

		if err != nil {
			return nil, err
		}

		return gin.H{}, nil
	})
}
