package service

import (
	"errors"
	"net/http"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/config"
	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/content"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// upsertLikeNotification 实现点赞通知的持久化与防刷屏。
// 返回非 nil 行表示应推送 notification 事件；被冷却抑制时返回 nil。
func upsertLikeNotification(
	tx *gorm.DB,
	recipientID, actorID uint,
	typ forummodels.NotificationType,
	threadID, replyID uint,
	excerpt string,
) (*forummodels.Notification, error) {
	if recipientID == 0 || recipientID == actorID {
		return nil, nil
	}

	now := time.Now()
	cooldown := time.Duration(config.Get().Forum.LikeNotificationCooldownMinutes) * time.Minute

	existing, err := dao.NotificationByKey(tx, typ, recipientID, actorID, threadID, replyID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return addNotification(tx, notificationInput{
			RecipientID: recipientID,
			ActorID:     actorID,
			Type:        typ,
			ThreadID:    threadID,
			ReplyID:     replyID,
			Excerpt:     excerpt,
		})
	}
	if err != nil {
		return nil, err
	}

	// 同一条通知在最近 X 分钟内创建且未读：抑制重复提醒。
	if existing.ReadAt == nil && cooldown > 0 && existing.CreatedAt.After(now.Add(-cooldown)) {
		return nil, nil
	}

	// 已读或超过窗口：刷新为新的未读通知，并保留同一 ID 供前端 upsert。
	if err := tx.Model(&forummodels.Notification{}).
		Where("id = ?", existing.ID).
		UpdateColumns(map[string]any{
			"created_at": now,
			"read_at":    nil,
			"excerpt":    excerpt,
		}).Error; err != nil {
		return nil, err
	}

	existing.CreatedAt = now
	existing.ReadAt = nil
	existing.Excerpt = excerpt
	return existing, nil
}

// LikeThread 点赞/取消点赞主题，返回当前状态与点赞数。
func LikeThread(user *models.User, threadID uint, like bool) (bool, int64, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	if _, err := requireEnabledBoard(thread.BoardID); err != nil {
		return false, 0, err
	}

	var (
		changed      bool
		notification *forummodels.Notification
	)
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if like {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.ThreadLike{
				ThreadID: thread.ID,
				UserID:   user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}

			changed = true
			n, err := upsertLikeNotification(
				tx,
				thread.AuthorID,
				user.ID,
				forummodels.NotificationTypeLikeThread,
				thread.ID,
				0,
				content.Excerpt(thread.ContentText, 160),
			)
			if err != nil {
				return err
			}
			notification = n
			return nil
		}

		result := tx.Where("thread_id = ? AND user_id = ?", thread.ID, user.ID).
			Delete(&forummodels.ThreadLike{})
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected > 0
		return nil
	}); err != nil {
		return false, 0, err
	}

	var count int64
	store.DB.Model(&forummodels.ThreadLike{}).Where("thread_id = ?", thread.ID).Count(&count)

	if changed {
		publishThreadLike(thread.ID, count, user.ID, like)
	}
	if notification != nil {
		publishNotifications([]forummodels.Notification{*notification})
	}
	return like, count, nil
}

// LikeReply 点赞/取消点赞跟帖。
func LikeReply(user *models.User, replyID uint, like bool) (bool, int64, error) {
	reply, err := dao.ReplyByID(replyID, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "跟帖不存在")
	}

	thread, err := dao.ThreadByID(reply.ThreadID, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "主题不存在")
	}
	if _, err := requireEnabledBoard(thread.BoardID); err != nil {
		return false, 0, err
	}

	var (
		changed      bool
		notification *forummodels.Notification
	)
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if like {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.ReplyLike{
				ReplyID: reply.ID,
				UserID:  user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}

			changed = true
			n, err := upsertLikeNotification(
				tx,
				reply.AuthorID,
				user.ID,
				forummodels.NotificationTypeLikeReply,
				reply.ThreadID,
				reply.ID,
				content.Excerpt(reply.ContentText, 160),
			)
			if err != nil {
				return err
			}
			notification = n
			return nil
		}

		result := tx.Where("reply_id = ? AND user_id = ?", reply.ID, user.ID).
			Delete(&forummodels.ReplyLike{})
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected > 0
		return nil
	}); err != nil {
		return false, 0, err
	}

	var count int64
	store.DB.Model(&forummodels.ReplyLike{}).Where("reply_id = ?", reply.ID).Count(&count)

	if changed {
		publishReplyLike(reply.ThreadID, reply.ID, count, user.ID, like)
	}
	if notification != nil {
		publishNotifications([]forummodels.Notification{*notification})
	}
	return like, count, nil
}

// LikeUser 点赞/取消点赞用户主页。
func LikeUser(user *models.User, targetUserID uint, like bool) (bool, int64, error) {
	if targetUserID == user.ID {
		return false, 0, h.HttpError(http.StatusBadRequest, "不能点赞自己")
	}

	var target models.User
	if err := store.DB.First(&target, targetUserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, 0, h.HttpError(http.StatusNotFound, "用户不存在")
		}
		return false, 0, err
	}

	var (
		changed      bool
		notification *forummodels.Notification
	)
	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if like {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.UserLike{
				TargetUserID: targetUserID,
				LikerID:      user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}

			changed = true
			n, err := upsertLikeNotification(
				tx,
				targetUserID,
				user.ID,
				forummodels.NotificationTypeLikeUser,
				0,
				0,
				"",
			)
			if err != nil {
				return err
			}
			notification = n
			return nil
		}

		result := tx.Where("target_user_id = ? AND liker_id = ?", targetUserID, user.ID).
			Delete(&forummodels.UserLike{})
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected > 0
		return nil
	}); err != nil {
		return false, 0, err
	}

	count := dao.UserLikeCount(targetUserID)
	if changed {
		publishUserLike(targetUserID, count, user.ID, like)
	}
	if notification != nil {
		publishNotifications([]forummodels.Notification{*notification})
	}
	return like, count, nil
}

// FollowBoard 关注/取消关注板块，返回关注状态与关注数。
func FollowBoard(user *models.User, slug string, follow bool) (bool, int64, error) {
	board, err := dao.BoardBySlug(slug, false, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "板块不存在")
	}

	var changed bool
	if follow {
		result := store.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.BoardFollow{
			BoardID: board.ID,
			UserID:  user.ID,
		})
		if result.Error != nil {
			return false, 0, result.Error
		}
		changed = result.RowsAffected > 0
	} else {
		result := store.DB.Where("board_id = ? AND user_id = ?", board.ID, user.ID).
			Delete(&forummodels.BoardFollow{})
		if result.Error != nil {
			return false, 0, result.Error
		}
		changed = result.RowsAffected > 0
	}

	counts := dao.BoardFollowerCounts([]uint{board.ID})
	count := counts[board.ID]
	if changed {
		publishBoardFollow(board.ID, board.Slug, count, user.ID, follow)
	}
	return follow, count, nil
}
