package user_routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/context_util"
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/monitors"
	"github.com/Subilan/go-aliyunmc/playerdata"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func HandleGetChatToken(c *gin.Context) (any, error) {
	user, exists := context_util.GetUser(c)

	if !exists {
		return nil, h.HttpError(http.StatusNotFound, "用户信息不存在")
	}
	
	serverStatus := monitors.GetServerStatusMonitor().Snapshot()

	if serverStatus.Error != nil {
		return nil, h.HttpError(http.StatusInternalServerError, "无法获取服务器状态")
	}

	if !serverStatus.Value.Online {
		return nil, h.HttpError(http.StatusBadRequest, "服务器当前离线，请稍候再试")
	}

	gameName := playerdata.LookupPlayerName(*user.WhitelistUUID)

	if gameName == "" {
		return nil, h.HttpError(http.StatusNotFound, "找不到玩家名称")
	}

	claims := jwt.MapClaims{
		"uuid": user.WhitelistUUID,
		"playername": gameName,
		"exp":  time.Now().Add(time.Duration(C.ChatToken.ExpireSeconds) * time.Second).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(C.ChatToken.Secret))
	if err != nil {
		return nil, fmt.Errorf("生成聊天令牌失败：%v", err)
	}

	return gin.H{"token": tokenString, "playername": gameName}, nil
}
