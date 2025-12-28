package get

import (
	"context"
	"net/http"
	"time"

	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	"github.com/gin-gonic/gin"
)

type StoredInstance struct {
	InstanceId   string     `json:"instanceId"`
	InstanceType string     `json:"instanceType"`
	RegionId     string     `json:"regionId"`
	ZoneId       string     `json:"zoneId"`
	Ip           *string    `json:"ip"`
	CreatedAt    time.Time  `json:"createdAt"`
	DeletedAt    *time.Time `json:"deletedAt"`
}

func Instance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "instanceId not provided"}
		}

		ctx, cancel := context.WithTimeout(c, 1*time.Second)
		defer cancel()

		var result StoredInstance

		err := globals.Pool.QueryRowContext(ctx, `
SELECT instance_id, instance_type, region_id, zone_id, ip, created_at, deleted_at FROM instances WHERE instance_id = ?
`, instanceId).Scan(&result.InstanceId, &result.InstanceType, &result.RegionId, &result.ZoneId, &result.Ip, &result.CreatedAt, &result.DeletedAt)

		if err != nil {
			return nil, err
		}

		return helpers.Data(result), nil
	})
}
