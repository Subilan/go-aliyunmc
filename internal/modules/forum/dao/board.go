package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
)

// BoardBySlug 按 slug 查找板块。默认只返回未删除的板块。
func BoardBySlug(slug string, includeDeleted bool, includeDisabled bool) (*forummodels.Board, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	if !includeDisabled {
		db = db.Where("enabled = ?", true)
	}

	var board forummodels.Board
	if err := db.Where("slug = ?", slug).First(&board).Error; err != nil {
		return nil, err
	}
	return &board, nil
}

// BoardByID 按 ID 查找板块。
func BoardByID(id uint, includeDeleted bool) (*forummodels.Board, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var board forummodels.Board
	if err := db.First(&board, id).Error; err != nil {
		return nil, err
	}
	return &board, nil
}

// EnabledBoards 返回启用且未删除的板块。
func EnabledBoards() ([]forummodels.Board, error) {
	var boards []forummodels.Board
	err := store.DB.Where("enabled = ?", true).Order("sort_order ASC, id ASC").Find(&boards).Error
	return boards, err
}

// BoardsByIDs 批量加载板块。
func BoardsByIDs(ids []uint, includeDeleted bool) map[uint]forummodels.Board {
	result := make(map[uint]forummodels.Board, len(ids))
	if len(ids) == 0 {
		return result
	}
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var boards []forummodels.Board
	if err := db.Where("id IN ?", ids).Find(&boards).Error; err != nil {
		return result
	}
	for _, b := range boards {
		result[b.ID] = b
	}
	return result
}

// BoardThreadCounts 返回板块的未删除主题数。
func BoardThreadCounts(boardIDs []uint) map[uint]int64 {
	return groupCount(boardIDs, "board_id", func(ids []uint) *gorm.DB {
		return store.DB.Model(&forummodels.Thread{}).Where("board_id IN ?", ids)
	})
}

// BoardFollowerIDs 返回板块关注者 ID，按 user_id 排序。
func BoardFollowerIDs(tx *gorm.DB, boardID uint) ([]uint, error) {
	var ids []uint
	err := tx.Model(&forummodels.BoardFollow{}).
		Where("board_id = ?", boardID).
		Order("user_id ASC").
		Pluck("user_id", &ids).Error
	return ids, err
}

// BoardFollowerCounts 返回板块关注数。
func BoardFollowerCounts(boardIDs []uint) map[uint]int64 {
	return groupCount(boardIDs, "board_id", func(ids []uint) *gorm.DB {
		return store.DB.Model(&forummodels.BoardFollow{}).Where("board_id IN ?", ids)
	})
}

// FollowedBoardIDs 返回用户在给定板块中关注了哪些。
func FollowedBoardIDs(userID uint, boardIDs []uint) map[uint]bool {
	result := make(map[uint]bool)
	if userID == 0 || len(boardIDs) == 0 {
		return result
	}

	var rows []forummodels.BoardFollow
	store.DB.Where("user_id = ? AND board_id IN ?", userID, boardIDs).Find(&rows)
	for _, row := range rows {
		result[row.BoardID] = true
	}
	return result
}
