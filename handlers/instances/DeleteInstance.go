package instances

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/gin-gonic/gin"
)

const DeleteInstanceTimeout = 15 * time.Second

type DeleteInstanceQuery struct {
	Force     bool `form:"force"`
	ForceStop bool `form:"forceStop"`
}

func HandleDeleteInstance() gin.HandlerFunc {
	return helpers.QueryHandler[DeleteInstanceQuery](func(query DeleteInstanceQuery, c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			err := db.Pool.QueryRow("SELECT instance_id FROM instances WHERE deleted_at IS NULL").Scan(&instanceId)

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

		err := db.Pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM instances WHERE deleted_at IS NULL AND instance_id = ?", instanceId).Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt == 0 {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "instance not found or already deleted"}
		}

		deleteInstanceRequest := &ecs20140526.DeleteInstanceRequest{
			InstanceId: &instanceId,
			Force:      &query.Force,
			ForceStop:  &query.ForceStop,
		}

		_, err = globals.EcsClient.DeleteInstance(deleteInstanceRequest)

		if err != nil {
			return nil, err
		}

		_, err = db.Pool.ExecContext(ctx, "UPDATE instances SET deleted_at = CURRENT_TIMESTAMP WHERE instance_id = ?", instanceId)

		if err != nil {
			return nil, err
		}

		// 将实例删除广播给所有用户
		event, err := store.BuildInstanceEvent(store.InstanceEventNotify, store.InstanceNotificationDeleted)

		if err != nil {
			log.Println("cannot build event:", err)
		}

		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast and save event:", err)
		}

		return gin.H{}, nil
	})
}
