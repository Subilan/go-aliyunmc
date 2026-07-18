package sample_routes

import (
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/store"
	"github.com/Subilan/go-aliyunmc/store/models"

	"github.com/gin-gonic/gin"
)

// PlayerOnlineRange 表示某个玩家的一次连续在线时段。
type PlayerOnlineRange struct {
	PlayerName string    `json:"playerName"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
}

func HandleGetPlayerOnlineRanges(c *gin.Context) (any, error) {
	// 解析时间范围，默认最近 6 小时
	now := time.Now()
	from := now.Add(-6 * time.Hour)
	to := now

	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return nil, h.HttpError(400, "from 参数格式无效，请使用 RFC3339")
		}
		from = t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return nil, h.HttpError(400, "to 参数格式无效，请使用 RFC3339")
		}
		to = t
	}

	// 查询时间范围内的所有采样记录，按时间升序
	var samples []models.PlayerListSample
	if err := store.DB.
		Where("created_at >= ? AND created_at <= ?", from, to).
		Order("created_at ASC").
		Find(&samples).Error; err != nil {
		return nil, err
	}

	if len(samples) == 0 {
		return []PlayerOnlineRange{}, nil
	}

	// 构建会话
	type partialSession struct {
		startAt    time.Time
		lastSeenAt time.Time
	}
	active := make(map[string]*partialSession)
	var ranges []PlayerOnlineRange

	for _, sample := range samples {
		currentPlayers := splitPlayerNames(sample.PlayerNames)

		// 检查之前在线但现在不在的玩家 → 结束会话
		for name, sess := range active {
			if !contains(currentPlayers, name) {
				ranges = append(ranges, PlayerOnlineRange{
					PlayerName: name,
					StartAt:    sess.startAt,
					EndAt:      sess.lastSeenAt,
				})
				delete(active, name)
			}
		}

		// 检查新上线或仍在线的玩家
		for _, name := range currentPlayers {
			if sess, ok := active[name]; ok {
				sess.lastSeenAt = sample.CreatedAt
			} else {
				active[name] = &partialSession{
					startAt:    sample.CreatedAt,
					lastSeenAt: sample.CreatedAt,
				}
			}
		}
	}

	// 关闭剩余活跃会话
	for name, sess := range active {
		ranges = append(ranges, PlayerOnlineRange{
			PlayerName: name,
			StartAt:    sess.startAt,
			EndAt:      sess.lastSeenAt,
		})
	}

	return ranges, nil
}

func splitPlayerNames(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
