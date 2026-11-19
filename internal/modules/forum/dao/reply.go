package dao

import (
	"database/sql"

	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
)

// ListThreadReplies 返回主题下的全部跟帖（含删除占位），按楼层升序。
// 同时返回归一化后的页码与页大小。
func ListThreadReplies(threadID uint, page, pageSize int) ([]forummodels.Reply, int64, int, int, error) {
	page, pageSize = Pagination(page, pageSize, 20, 50)

	db := store.DB.Unscoped().Model(&forummodels.Reply{}).Where("thread_id = ?", threadID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	var replies []forummodels.Reply
	err := db.Order("floor ASC").
		Offset(Offset(page, pageSize)).
		Limit(pageSize).
		Find(&replies).Error
	if err != nil {
		return nil, 0, page, pageSize, err
	}

	return replies, total, page, pageSize, nil
}

// ReplyFilter 是后台跟帖列表的查询条件。
type ReplyFilter struct {
	Status            string // active / deleted / all
	ThreadID          uint
	ThreadIDs         []uint
	BoardIDs          []uint
	AuthorID          uint
	Keyword           string
	OnlyEnabledBoards bool
	Page              int
	PageSize          int
	// MaxPageSize 是接口允许的最大页大小；0 表示后台默认 100。
	MaxPageSize int
}

// ListReplies 返回后台跟帖分页结果，以及归一化后的页码与页大小。
func ListReplies(f ReplyFilter) ([]forummodels.Reply, int64, int, int, error) {
	maxSize := 100
	if f.MaxPageSize > 0 {
		maxSize = f.MaxPageSize
	}
	page, pageSize := Pagination(f.Page, f.PageSize, 20, maxSize)

	db := store.DB.Model(&forummodels.Reply{})
	joinThreads := false
	if f.OnlyEnabledBoards {
		joinThreads = true
		db = db.
			Joins("JOIN forum_threads ON forum_threads.id = forum_replies.thread_id AND forum_threads.deleted_at IS NULL").
			Joins("JOIN forum_boards ON forum_boards.id = forum_threads.board_id AND forum_boards.deleted_at IS NULL AND forum_boards.enabled = 1")
	}
	switch f.Status {
	case "all":
		db = db.Unscoped()
	case "deleted":
		db = db.Unscoped().Where("forum_replies.deleted_at IS NOT NULL")
	default:
		db = db.Where("forum_replies.deleted_at IS NULL")
	}

	if f.ThreadID != 0 {
		db = db.Where("forum_replies.thread_id = ?", f.ThreadID)
	}
	if len(f.ThreadIDs) > 0 {
		db = db.Where("forum_replies.thread_id IN ?", f.ThreadIDs)
	}
	if len(f.BoardIDs) > 0 {
		if !joinThreads {
			db = db.Joins("JOIN forum_threads ON forum_threads.id = forum_replies.thread_id")
		}
		db = db.Where("forum_threads.board_id IN ?", f.BoardIDs)
	}
	if f.AuthorID != 0 {
		db = db.Where("forum_replies.author_id = ?", f.AuthorID)
	}
	if f.Keyword != "" {
		db = db.Where("forum_replies.content_text LIKE ?", "%"+f.Keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	var replies []forummodels.Reply
	if joinThreads || len(f.BoardIDs) > 0 {
		db = db.Select("forum_replies.*")
	}
	err := db.Order("forum_replies.thread_id ASC, forum_replies.floor ASC").
		Offset(Offset(page, pageSize)).
		Limit(pageSize).
		Find(&replies).Error
	if err != nil {
		return nil, 0, page, pageSize, err
	}

	return replies, total, page, pageSize, nil
}

// ReplyByID 读取跟帖。includeDeleted 为 true 时包含软删除跟帖。
func ReplyByID(id uint, includeDeleted bool) (*forummodels.Reply, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var reply forummodels.Reply
	if err := db.First(&reply, id).Error; err != nil {
		return nil, err
	}
	return &reply, nil
}

// MaxFloor 返回主题当前最大楼层（包含已删除回复）。
func MaxFloor(tx *gorm.DB, threadID uint) (uint, error) {
	var maxFloor uint
	err := tx.Unscoped().Model(&forummodels.Reply{}).
		Where("thread_id = ?", threadID).
		Select("COALESCE(MAX(floor), 0)").
		Scan(&maxFloor).Error
	return maxFloor, err
}

// ReplyByFloor 查找主题内指定楼层的回复（包含已删除）。
func ReplyByFloor(tx *gorm.DB, threadID, floor uint) (*forummodels.Reply, error) {
	var reply forummodels.Reply
	err := tx.Unscoped().Where("thread_id = ? AND floor = ?", threadID, floor).First(&reply).Error
	if err != nil {
		return nil, err
	}
	return &reply, nil
}

// ReplyLikeCounts 返回跟帖点赞数。
func ReplyLikeCounts(replyIDs []uint) map[uint]int64 {
	return groupCount(replyIDs, "reply_id", func(ids []uint) *gorm.DB {
		return store.DB.Model(&forummodels.ReplyLike{}).Where("reply_id IN ?", ids)
	})
}

// LikedReplyIDs 返回用户点赞过的跟帖集合。
func LikedReplyIDs(userID uint, replyIDs []uint) map[uint]bool {
	result := make(map[uint]bool)
	if userID == 0 || len(replyIDs) == 0 {
		return result
	}
	var rows []forummodels.ReplyLike
	store.DB.Where("user_id = ? AND reply_id IN ?", userID, replyIDs).Find(&rows)
	for _, row := range rows {
		result[row.ReplyID] = true
	}
	return result
}

// RecalculateLastReplyAt 重算主题最近回复时间。
// 主题下已无未删除跟帖时 MAX(created_at) 为 NULL，必须用 sql.NullTime 承接，
// 否则 MySQL 驱动会返回扫描错误。
func RecalculateLastReplyAt(tx *gorm.DB, threadID uint) error {
	var last sql.NullTime
	if err := tx.Model(&forummodels.Reply{}).
		Where("thread_id = ?", threadID).
		Select("MAX(created_at)").
		Scan(&last).Error; err != nil {
		return err
	}

	var value any
	if last.Valid {
		value = last.Time
	}

	return tx.Model(&forummodels.Thread{}).
		Where("id = ?", threadID).
		Update("last_reply_at", value).Error
}
