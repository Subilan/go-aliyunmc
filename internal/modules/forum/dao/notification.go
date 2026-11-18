package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
)

// NotificationFilter 是通知列表的查询条件。
type NotificationFilter struct {
	RecipientID uint
	UnreadOnly  bool
	Page        int
	PageSize    int
}

// ListNotifications 返回通知分页结果，以及归一化后的页码与页大小。
func ListNotifications(f NotificationFilter) ([]forummodels.Notification, int64, int, int, error) {
	page, pageSize := Pagination(f.Page, f.PageSize, 20, 50)

	db := store.DB.Model(&forummodels.Notification{}).Where("recipient_id = ?", f.RecipientID)
	if f.UnreadOnly {
		db = db.Where("read_at IS NULL")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, err
	}

	var notifications []forummodels.Notification
	err := db.Order("created_at DESC, id DESC").
		Offset(Offset(page, pageSize)).
		Limit(pageSize).
		Find(&notifications).Error
	if err != nil {
		return nil, 0, page, pageSize, err
	}

	return notifications, total, page, pageSize, nil
}

// UnreadNotificationCount 返回未读通知数。
func UnreadNotificationCount(userID uint) int64 {
	var count int64
	store.DB.Model(&forummodels.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", userID).
		Count(&count)
	return count
}

// ThreadTitles 批量加载主题标题。
func ThreadTitles(threadIDs []uint) map[uint]string {
	titles := make(map[uint]string, len(threadIDs))
	if len(threadIDs) == 0 {
		return titles
	}
	var threads []forummodels.Thread
	store.DB.Unscoped().Select("id, title").Where("id IN ?", threadIDs).Find(&threads)
	for _, t := range threads {
		titles[t.ID] = t.Title
	}
	return titles
}
