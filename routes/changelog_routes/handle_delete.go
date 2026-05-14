package changelog_routes

import (
	"errors"
	"github.com/Subilan/go-aliyunmc/store"
	"strconv"

	"github.com/gin-gonic/gin"
)

func HandleDelete(c *gin.Context) (any, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, errors.New("无效的 id")
	}

	if err := store.DeleteChangelog(uint(id)); err != nil {
		return nil, err
	}

	return nil, nil
}
