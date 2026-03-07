package server

import (
	"fmt"
	"strings"

	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/gin-gonic/gin"
)

type GetPlayTimeRankingQuery struct {
	helpers.Paginated
	helpers.Sorted
}

type GetPlayTimeRankingResponse struct {
	Items []store.PlayTimeData `json:"items"`
	Total int                  `json:"total"`
}

// HandleGetPlayTimeRanking 获取游戏时间排行榜
//
//	@Summary		获取游戏时间排行榜
//	@Description	从 play_time.db 中查询所有参与排行榜的玩家数据，支持分页和排序
//	@Tags			server
//	@Produce		json
//	@Param			page		query	int		false	"页码"
//	@Param			pageSize	query	int		false	"每页数量"
//	@Param			sortBy		query	string	false	"排序字段 (playTime/lastSeen/nickname)"
//	@Param			sortOrder	query	string	false	"排序顺序 (asc/desc)"
//	@Success		200			{object}	helpers.DataResp[GetPlayTimeRankingResponse]
//	@Failure		500			{object}	helpers.ErrorResp
//	@Router			/server/play-time-ranking [get]
func HandleGetPlayTimeRanking() gin.HandlerFunc {
	return helpers.QueryHandler(func(query GetPlayTimeRankingQuery, c *gin.Context) (any, error) {
		page := query.Page
		pageSize := query.PageSize
		sortBy := query.SortBy
		sortOrder := query.SortOrder

		result, err := getPlayTimeRanking(page, pageSize, sortBy, sortOrder)
		if err != nil {
			return nil, err
		}

		return helpers.Data(result), nil
	})
}

// getPlayTimeRanking 获取游戏时间排行榜（私有函数）
// 会过滤掉 user_preferences 中 participate_in_play_time_ranking 为 false 的用户
func getPlayTimeRanking(page, pageSize int, sortBy, sortOrder *string) (*GetPlayTimeRankingResponse, error) {
	// 1. 从 MySQL 查询不参与排行榜的用户名列表
	excludedNicknames, err := getExcludedNicknames()
	if err != nil {
		return nil, fmt.Errorf("get excluded nicknames: %w", err)
	}

	// 2. 从 SQLite 查询排行榜数据
	sqliteDb, err := store.GetPlayTimeDB()
	if err != nil {
		return nil, err
	}
	defer sqliteDb.Close()

	// 计算分页
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// 构建排序子句
	orderClause := buildOrderClause(sortBy, sortOrder)

	// 构建 WHERE 子句和参数
	whereClause, args := buildWhereClause(excludedNicknames)

	// 计算总数
	var total int
	err = sqliteDb.QueryRow(`
		SELECT COUNT(*)
		FROM play_time pt
		`+whereClause,
		args...,
	).Scan(&total)

	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	// 查询数据
	rows, err := sqliteDb.Query(`
		SELECT uuid, nickname, playtime, artificial_playtime, afk_playtime,
		       last_seen, first_join, relative_join_streak, absolute_join_streak
		FROM play_time pt
		`+whereClause+`
		`+orderClause+`
		LIMIT ? OFFSET ?
	`, append(args, pageSize, offset)...)

	if err != nil {
		return nil, fmt.Errorf("query data: %w", err)
	}
	defer rows.Close()

	var items []store.PlayTimeData
	for rows.Next() {
		var item store.PlayTimeData
		err := rows.Scan(
			&item.UUID,
			&item.Nickname,
			&item.PlayTime,
			&item.ArtificialPlayTime,
			&item.AfkPlayTime,
			&item.LastSeen,
			&item.FirstJoin,
			&item.RelativeJoinStreak,
			&item.AbsoluteJoinStreak,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &GetPlayTimeRankingResponse{
		Items: items,
		Total: total,
	}, nil
}

// buildWhereClause 构建 WHERE 子句和参数列表
func buildWhereClause(excludedNicknames []string) (string, []interface{}) {
	if len(excludedNicknames) == 0 {
		return "", nil
	}

	placeholders := make([]string, len(excludedNicknames))
	args := make([]interface{}, len(excludedNicknames))
	for i, nickname := range excludedNicknames {
		placeholders[i] = "?"
		args[i] = nickname
	}

	whereClause := "WHERE nickname NOT IN (" + strings.Join(placeholders, ",") + ")"
	return whereClause, args
}

// getExcludedNicknames 从 MySQL 查询不参与排行榜的用户名列表
func getExcludedNicknames() ([]string, error) {
	rows, err := db.Pool.Query(`
		SELECT username
		FROM user_preferences
		WHERE preference_key = 'participate_in_play_time_ranking'
		AND preference_value = 'false'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nicknames []string
	for rows.Next() {
		var nickname string
		if err := rows.Scan(&nickname); err != nil {
			return nil, err
		}
		nicknames = append(nicknames, nickname)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nicknames, nil
}

// buildOrderClause 构建排序子句
func buildOrderClause(sortBy, sortOrder *string) string {
	// 默认的排序字段和顺序
	defaultSortField := "playtime"
	defaultSortOrder := "DESC"

	// 如果没有指定排序字段，使用默认排序
	if sortBy == nil || *sortBy == "" {
		return fmt.Sprintf("ORDER BY %s %s", defaultSortField, defaultSortOrder)
	}

	// 映射前端传来的字段名到数据库字段名（只允许 playTime、lastSeen 和 nickname）
	fieldMapping := map[string]string{
		"playTime": "playtime",
		"lastSeen": "last_seen",
		"nickname": "nickname",
	}

	// 获取数据库字段名
	dbField, ok := fieldMapping[*sortBy]
	if !ok {
		// 无效的排序字段，使用默认排序
		return fmt.Sprintf("ORDER BY %s %s", defaultSortField, defaultSortOrder)
	}

	// 确定排序顺序
	order := defaultSortOrder
	if sortOrder != nil {
		switch *sortOrder {
		case "asc", "ASC":
			order = "ASC"
		case "desc", "DESC":
			order = "DESC"
		default:
			order = defaultSortOrder
		}
	}

	return fmt.Sprintf("ORDER BY %s %s", dbField, order)
}
