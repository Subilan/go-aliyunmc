package changelog_routes

import (
	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/store"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func HandleLike(c *gin.Context) (any, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, h.HttpError(http.StatusNotFound, "无效的 id")
	}

	userID, ok := context_util.GetUserID(c)
	if !ok {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	liked, likeCount, err := store.ToggleLike(uint(id), userID)
	if err != nil {
		return nil, err
	}

	return gin.H{"liked": liked, "like_count": likeCount}, nil
}
