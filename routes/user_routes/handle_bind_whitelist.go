package user_routes

import (
	"errors"
	"net/http"

	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/playerdata"
	"go-aliyunmc/store"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type bindWhitelistRequest struct {
	Name string `json:"name" binding:"required"`
}

// HandleBindWhitelist 将当前用户与一个白名单项目绑定，按 Mojang 游戏名查找对应 UUID 并存储
func HandleBindWhitelist(req bindWhitelistRequest, c *gin.Context) (any, error) {
	entries, err := playerdata.ReadWhitelist()
	if err != nil {
		return nil, h.HttpError(http.StatusInternalServerError, "读取白名单失败")
	}

	var matched *playerdata.WhitelistEntry
	for i := range entries {
		if entries[i].Name == req.Name {
			matched = &entries[i]
			break
		}
	}
	if matched == nil {
		return nil, h.HttpError(http.StatusNotFound, "玩家不存在")
	}

	user, exists := context_util.GetUser(c)
	if !exists {
		return nil, h.HttpError(http.StatusUnauthorized, "未登录")
	}

	user.WhitelistUUID = &matched.UUID

	if err := store.DB.Save(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, h.HttpError(http.StatusConflict, "该游戏名已被绑定")
		}
		return nil, err
	}

	return nil, nil
}
