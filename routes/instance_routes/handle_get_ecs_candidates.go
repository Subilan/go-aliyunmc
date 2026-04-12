package instance_routes

import (
	"go-aliyunmc/global_states"

	"github.com/gin-gonic/gin"
)

func HandleGetEcsCandidates(c *gin.Context) (any, error) {
	return global_states.GetCurrentEcsCandidates(), nil
}