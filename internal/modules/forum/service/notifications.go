package service

import (
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
)

// ListNotifications 返回当前用户的通知分页。
func ListNotifications(user *models.User, filter string, page, pageSize int) (dto.Page[dto.Notification], error) {
	rows, total, page, pageSize, err := dao.ListNotifications(dao.NotificationFilter{
		RecipientID: user.ID,
		UnreadOnly:  filter == "unread",
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		return dto.Page[dto.Notification]{}, err
	}

	actorIDs := make([]uint, 0, len(rows))
	threadIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		actorIDs = append(actorIDs, row.ActorID)
		threadIDs = append(threadIDs, row.ThreadID)
	}
	actorIDs = uniqueIDs(actorIDs)
	threadIDs = uniqueIDs(threadIDs)

	views := dao.LoadUserViews(actorIDs)
	titles := dao.ThreadTitles(threadIDs)

	return dto.NewPage(dto.Notifications(rows, views, titles), total, page, pageSize), nil
}

// UnreadCount 返回未读通知数。
func UnreadCount(user *models.User) int64 {
	return dao.UnreadNotificationCount(user.ID)
}

// maxNotificationIDs 限制单次标记已读的通知数量，避免超长 IN 查询。
const maxNotificationIDs = 100

// MarkNotificationsRead 标记单条或全部通知已读，返回剩余未读数。
func MarkNotificationsRead(user *models.User, ids []uint, all bool) (int64, error) {
	if !all {
		if len(ids) == 0 {
			return 0, h.HttpError(http.StatusBadRequest, "请提供要标记的通知 ID 或 all=true")
		}
		if len(ids) > maxNotificationIDs {
			return 0, h.HttpError(http.StatusBadRequest, "单次最多标记 100 条通知")
		}
	}

	now := time.Now()
	db := store.DB.Model(&forummodels.Notification{}).
		Where("recipient_id = ? AND read_at IS NULL", user.ID)
	if !all {
		db = db.Where("id IN ?", ids)
	}

	if err := db.Update("read_at", now).Error; err != nil {
		return 0, err
	}

	return dao.UnreadNotificationCount(user.ID), nil
}
