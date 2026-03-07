package server

import (
	"fmt"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type GetPlayTimeOverviewResponse struct {
	TotalPlayTime        int64  `json:"totalPlayTime"`
	LatestPlayerName     string `json:"latestPlayerName"`
	LatestPlayerLastSeen int64  `json:"latestPlayerLastSeen"`
}

// HandleGetPlayTimeOverview 获取游戏时间概览指标
//
//	@Summary		获取游戏时间概览
//	@Description	返回总游玩时间、最近在线玩家等概览指标
//	@Tags			server
//	@Produce		json
//	@Success		200		{object}	helpers.DataResp[GetPlayTimeOverviewResponse]
//	@Failure		500		{object}	helpers.ErrorResp
//	@Router			/server/play-time-overview [get]
func HandleGetPlayTimeOverview() gin.HandlerFunc {
	return helpers.BasicHandler(func(c *gin.Context) (any, error) {
		result, err := getPlayTimeOverview()
		if err != nil {
			return nil, err
		}

		return helpers.Data(result), nil
	})
}

// getPlayTimeOverview 获取游戏时间概览指标（私有函数）
func getPlayTimeOverview() (*GetPlayTimeOverviewResponse, error) {
	db, err := store.GetPlayTimeDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// 1. 查询所有玩家游玩时间的加和
	var totalPlayTime int64
	err = db.QueryRow(`
		SELECT SUM(playtime)
		FROM play_time
	`).Scan(&totalPlayTime)

	if err != nil {
		return nil, fmt.Errorf("query total play time: %w", err)
	}

	// 2. 查询最近在线的玩家（lastSeen 最大的玩家）
	var latestPlayerName string
	var latestPlayerLastSeen int64
	err = db.QueryRow(`
		SELECT nickname, last_seen
		FROM play_time
		ORDER BY last_seen DESC
		LIMIT 1
	`).Scan(&latestPlayerName, &latestPlayerLastSeen)

	if err != nil {
		// 如果没有记录，返回空值
		if err.Error() == "sql: no rows in result set" {
			return &GetPlayTimeOverviewResponse{
				TotalPlayTime:        totalPlayTime,
				LatestPlayerName:     "",
				LatestPlayerLastSeen: 0,
			}, nil
		}
		return nil, fmt.Errorf("query latest player: %w", err)
	}

	return &GetPlayTimeOverviewResponse{
		TotalPlayTime:        totalPlayTime,
		LatestPlayerName:     latestPlayerName,
		LatestPlayerLastSeen: latestPlayerLastSeen,
	}, nil
}
