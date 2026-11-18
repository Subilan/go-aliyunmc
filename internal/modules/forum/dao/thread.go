package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ThreadFilter 是主题列表的查询条件。公开接口与后台接口共用。
type ThreadFilter struct {
	Status            string // active / deleted / all；空等于 active
	BoardID           uint
	BoardIDs          []uint
	AuthorID          uint
	Pinned            *bool
	Locked            *bool
	Keyword           string
	OnlyEnabledBoards bool
	Sort              string // created / last_reply
	Page              int
	PageSize          int
	// MaxPageSize 是接口允许的最大页大小；0 表示后台默认 100。
	MaxPageSize int
}

// ListThreads 返回主题分页结果，以及归一化后的页码与页大小。
func ListThreads(f ThreadFilter) ([]forummodels.Thread, int64, int, int, error) {
	maxSize := 100
	if f.MaxPageSize > 0 {
		maxSize = f.MaxPageSize
	}
	page, pageSize := Pagination(f.Page, f.PageSize, 20, maxSize)

	db := store.DB.Model(&forummodels.Thread{})

	switch f.Status {
	case "all":
		db = db.Unscoped()
	case "deleted":
		db = db.Unscoped().Where("forum_threads.deleted_at IS NOT NULL")
	default:
		db = db.Where("forum_threads.deleted_at IS NULL")
	}

	if f.BoardID != 0 {
		db = db.Where("forum_threads.board_id = ?", f.BoardID)
	}
	if len(f.BoardIDs) > 0 {
		db = db.Where("forum_threads.board_id IN ?", f.BoardIDs)
	}
	if f.AuthorID != 0 {
		db = db.Where("forum_threads.author_id = ?", f.AuthorID)
	}
	if f.Pinned != nil {
		db = db.Where("forum_threads.pinned = ?", *f.Pinned)
	}
	if f.Locked != nil {
		db = db.Where("forum_threads.locked = ?", *f.Locked)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("forum_threads.title LIKE ? OR forum_threads.content_text LIKE ?", like, like)
	}
	if f.OnlyEnabledBoards {
		db = db.Joins("JOIN forum_boards ON forum_boards.id = forum_threads.board_id AND forum_boards.deleted_at IS NULL AND forum_boards.enabled = 1")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	order := "forum_threads.created_at DESC, forum_threads.id DESC"
	switch f.Sort {
	case "last_reply":
		order = "forum_threads.last_reply_at DESC, forum_threads.created_at DESC"
	case "pinned":
		order = "forum_threads.pinned_at DESC, forum_threads.created_at DESC, forum_threads.id DESC"
	}

	var threads []forummodels.Thread
	err := db.Order(order).
		Offset(Offset(page, pageSize)).
		Limit(pageSize).
		Find(&threads).Error
	if err != nil {
		return nil, 0, page, pageSize, err
	}

	return threads, total, page, pageSize, nil
}

// ThreadByID 读取主题。includeDeleted 为 true 时连软删除主题一起读取。
func ThreadByID(id uint, includeDeleted bool) (*forummodels.Thread, error) {
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var thread forummodels.Thread
	if err := db.First(&thread, id).Error; err != nil {
		return nil, err
	}
	return &thread, nil
}

// ThreadByIDForUpdate 在事务内锁定主题行，用于回复楼层分配。
func ThreadByIDForUpdate(tx *gorm.DB, id uint) (*forummodels.Thread, error) {
	var thread forummodels.Thread
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&thread, id).Error
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// ThreadsByIDs 批量加载主题。
func ThreadsByIDs(ids []uint, includeDeleted bool) map[uint]forummodels.Thread {
	result := make(map[uint]forummodels.Thread, len(ids))
	if len(ids) == 0 {
		return result
	}
	db := store.DB
	if includeDeleted {
		db = db.Unscoped()
	}
	var threads []forummodels.Thread
	if err := db.Where("id IN ?", ids).Find(&threads).Error; err != nil {
		return result
	}
	for _, t := range threads {
		result[t.ID] = t
	}
	return result
}

// ReplyCounts 返回主题的未删除回复数。
func ReplyCounts(threadIDs []uint) map[uint]int64 {
	return groupCount(threadIDs, "thread_id", func(ids []uint) *gorm.DB {
		return store.DB.Model(&forummodels.Reply{}).Where("thread_id IN ?", ids)
	})
}

// ThreadLikeCounts 返回主题点赞数。
func ThreadLikeCounts(threadIDs []uint) map[uint]int64 {
	return groupCount(threadIDs, "thread_id", func(ids []uint) *gorm.DB {
		return store.DB.Model(&forummodels.ThreadLike{}).Where("thread_id IN ?", ids)
	})
}

// LikedThreadIDs 返回用户点赞过的主题集合。
func LikedThreadIDs(userID uint, threadIDs []uint) map[uint]bool {
	result := make(map[uint]bool)
	if userID == 0 || len(threadIDs) == 0 {
		return result
	}
	var rows []forummodels.ThreadLike
	store.DB.Where("user_id = ? AND thread_id IN ?", userID, threadIDs).Find(&rows)
	for _, row := range rows {
		result[row.ThreadID] = true
	}
	return result
}

// LoadThreadTags 批量加载主题标签名。
func LoadThreadTags(threadIDs []uint) map[uint][]string {
	result := make(map[uint][]string, len(threadIDs))
	if len(threadIDs) == 0 {
		return result
	}

	type row struct {
		ThreadID uint
		Name     string
	}
	var rows []row
	store.DB.Model(&forummodels.ThreadTag{}).
		Select("forum_thread_tags.thread_id, forum_tags.name").
		Joins("JOIN forum_tags ON forum_tags.id = forum_thread_tags.tag_id").
		Where("forum_thread_tags.thread_id IN ?", threadIDs).
		Order("forum_tags.name ASC").
		Scan(&rows)

	for _, r := range rows {
		result[r.ThreadID] = append(result[r.ThreadID], r.Name)
	}
	return result
}

// groupCount 是按某一列批量计数的通用实现。
func groupCount(ids []uint, column string, query func([]uint) *gorm.DB) map[uint]int64 {
	counts := make(map[uint]int64, len(ids))
	if len(ids) == 0 {
		return counts
	}

	type row struct {
		Key   uint
		Count int64
	}
	var rows []row
	query(ids).
		Select(column + " AS `key`, COUNT(*) AS count").
		Group(column).
		Scan(&rows)

	for _, r := range rows {
		counts[r.Key] = r.Count
	}
	return counts
}
