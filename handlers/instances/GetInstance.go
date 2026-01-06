package instances

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

func HandleGetInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		instanceId := c.Param("instanceId")

		if instanceId == "" {
			return nil, &helpers.HttpError{Code: http.StatusBadRequest, Details: "instanceId not provided"}
		}

		var result store.Instance

		err := db.Pool.QueryRow(`
SELECT instance_id, instance_type, region_id, zone_id, ip, created_at, deleted_at FROM instances WHERE instance_id = ?
`, instanceId).Scan(&result.InstanceId, &result.InstanceType, &result.RegionId, &result.ZoneId, &result.Ip, &result.CreatedAt, &result.DeletedAt)

		if err != nil {
			return nil, err
		}

		return helpers.Data(result), nil
	})
}

func HandleGetActiveOrLatestInstance() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		activeInstance := store.GetActiveInstance()

		if activeInstance == nil {
			latestInstance := store.GetLatestInstance()

			if latestInstance == nil {
				return helpers.Data(gin.H{"instance": gin.H{}, "status": gin.H{}}), nil
			}

			return helpers.Data(gin.H{"instance": latestInstance, "status": gin.H{}}), nil
		}

		return helpers.Data(gin.H{"instance": activeInstance, "status": store.GetInstanceStatus(activeInstance.InstanceId)}), nil
	})
}
