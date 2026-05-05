package user_routes

import (
	"encoding/json"
	"net/http"
	"os"

	"go-aliyunmc/context_util"
	"go-aliyunmc/h"
	"go-aliyunmc/store"

	"github.com/gin-gonic/gin"
)

const whitelistPath = "remote_data_cache/whitelist.json"

type whitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type bindWhitelistRequest struct {
	Name string `json:"name" binding:"required"`
}

func readWhitelist() ([]whitelistEntry, error) {
	data, err := os.ReadFile(whitelistPath)
	if err != nil {
		return nil, err
	}
	var entries []whitelistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// HandleBindWhitelist 将当前用户与一个白名单项目绑定，按 Mojang 游戏名查找对应 UUID 并存储
func HandleBindWhitelist(req bindWhitelistRequest, c *gin.Context) (any, error) {
	entries, err := readWhitelist()
	if err != nil {
		return nil, h.HttpError(http.StatusInternalServerError, "读取白名单失败")
	}

	var matched *whitelistEntry
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
		return nil, err
	}

	return nil, nil
}
