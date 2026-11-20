package dao

import (
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// ThreadTitles 批量加载主题标题。thread_id = 0 的站内通知（如主页点赞）不查询。
func ThreadTitles(threadIDs []uint) map[uint]string {
	titles := make(map[uint]string, len(threadIDs))
	ids := make([]uint, 0, len(threadIDs))
	for _, id := range threadIDs {
		if id != 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return titles
	}
	var threads []forummodels.Thread
	store.DB.Unscoped().Select("id, title").Where("id IN ?", ids).Find(&threads)
	for _, t := range threads {
		titles[t.ID] = t.Title
	}
	return titles
}

// UnreadNotificationCounts 批量返回一组用户的未读通知数。
func UnreadNotificationCounts(userIDs []uint) map[uint]int64 {
	counts := make(map[uint]int64, len(userIDs))
	ids := uniqueIDs(userIDs)
	if len(ids) == 0 {
		return counts
	}

	const chunkSize = 500
	type row struct {
		RecipientID uint
		Count       int64
	}
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}

		var rows []row
		store.DB.Model(&forummodels.Notification{}).
			Select("recipient_id, COUNT(*) AS count").
			Where("recipient_id IN ? AND read_at IS NULL", ids[start:end]).
			Group("recipient_id").
			Scan(&rows)
		for _, item := range rows {
			counts[item.RecipientID] = item.Count
		}
	}
	return counts
}

// HasContentNotification 判断某接收者是否已有一条内容类通知（reply/quote/mention）。
// 数据库唯一索引按 type 区分，跨类型去重必须依赖该查询。
func HasContentNotification(tx *gorm.DB, recipientID, threadID, replyID uint) (bool, error) {
	var count int64
	err := tx.Model(&forummodels.Notification{}).
		Where("recipient_id = ? AND thread_id = ? AND reply_id = ? AND type IN ?",
			recipientID, threadID, replyID,
			[]string{
				string(forummodels.NotificationTypeReply),
				string(forummodels.NotificationTypeQuote),
				string(forummodels.NotificationTypeMention),
			}).
		Count(&count).Error
	return count > 0, err
}

// NotificationByKey 在事务内按唯一键读取通知并加行锁。
func NotificationByKey(
	tx *gorm.DB,
	typ forummodels.NotificationType,
	recipientID, actorID, threadID, replyID uint,
) (*forummodels.Notification, error) {
	var row forummodels.Notification
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("type = ? AND recipient_id = ? AND actor_id = ? AND thread_id = ? AND reply_id = ?",
			string(typ), recipientID, actorID, threadID, replyID).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func uniqueIDs(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
