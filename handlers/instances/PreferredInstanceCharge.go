package instances

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/gin-gonic/gin"
)

func HandleGetPreferredInstanceCharge() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		chargePresent := monitors.SnapshotPreferredInstanceChargePresent()

		if !chargePresent {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "暂无符合要求的实例信息"}
		}

		charge := monitors.SnapshotPreferredInstanceCharge()

		return helpers.Data(charge), nil
	})
}
