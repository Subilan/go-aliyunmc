package service

import (
	"errors"
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/content"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LikeThread 点赞/取消点赞主题，返回当前状态与点赞数。
func LikeThread(user *models.User, threadID uint, like bool) (bool, int64, error) {
	thread, err := dao.ThreadByID(threadID, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "主题不存在")
	}

	if _, err := requireEnabledBoard(thread.BoardID); err != nil {
		return false, 0, err
	}

	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if like {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.ThreadLike{
				ThreadID: thread.ID,
				UserID:   user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 && thread.AuthorID != user.ID {
				return addNotification(tx, thread.AuthorID, user.ID, forummodels.NotificationTypeLikeThread, thread.ID, 0, content.Excerpt(thread.ContentText, 160))
			}
			return nil
		}

		if err := tx.Where("thread_id = ? AND user_id = ?", thread.ID, user.ID).
			Delete(&forummodels.ThreadLike{}).Error; err != nil {
			return err
		}
		return tx.Where(
			"type = ? AND actor_id = ? AND recipient_id = ? AND thread_id = ? AND reply_id = 0",
			forummodels.NotificationTypeLikeThread, user.ID, thread.AuthorID, thread.ID,
		).Delete(&forummodels.Notification{}).Error
	}); err != nil {
		return false, 0, err
	}

	var count int64
	store.DB.Model(&forummodels.ThreadLike{}).Where("thread_id = ?", thread.ID).Count(&count)
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

	if err := store.DB.Transaction(func(tx *gorm.DB) error {
		if like {
			result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.ReplyLike{
				ReplyID: reply.ID,
				UserID:  user.ID,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 && reply.AuthorID != user.ID {
				return addNotification(tx, reply.AuthorID, user.ID, forummodels.NotificationTypeLikeReply, reply.ThreadID, reply.ID, content.Excerpt(reply.ContentText, 160))
			}
			return nil
		}

		if err := tx.Where("reply_id = ? AND user_id = ?", reply.ID, user.ID).
			Delete(&forummodels.ReplyLike{}).Error; err != nil {
			return err
		}
		return tx.Where(
			"type = ? AND actor_id = ? AND recipient_id = ? AND thread_id = ? AND reply_id = ?",
			forummodels.NotificationTypeLikeReply, user.ID, reply.AuthorID, reply.ThreadID, reply.ID,
		).Delete(&forummodels.Notification{}).Error
	}); err != nil {
		return false, 0, err
	}

	var count int64
	store.DB.Model(&forummodels.ReplyLike{}).Where("reply_id = ?", reply.ID).Count(&count)
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

	if like {
		if err := store.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.UserLike{
			TargetUserID: targetUserID,
			LikerID:      user.ID,
		}).Error; err != nil {
			return false, 0, err
		}
	} else {
		if err := store.DB.Where("target_user_id = ? AND liker_id = ?", targetUserID, user.ID).
			Delete(&forummodels.UserLike{}).Error; err != nil {
			return false, 0, err
		}
	}

	return like, dao.UserLikeCount(targetUserID), nil
}

// FollowBoard 关注/取消关注板块，返回关注状态与关注数。
func FollowBoard(user *models.User, slug string, follow bool) (bool, int64, error) {
	board, err := dao.BoardBySlug(slug, false, false)
	if err != nil {
		return false, 0, h.HttpError(http.StatusNotFound, "板块不存在")
	}

	if follow {
		if err := store.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&forummodels.BoardFollow{
			BoardID: board.ID,
			UserID:  user.ID,
		}).Error; err != nil {
			return false, 0, err
		}
	} else {
		if err := store.DB.Where("board_id = ? AND user_id = ?", board.ID, user.ID).
			Delete(&forummodels.BoardFollow{}).Error; err != nil {
			return false, 0, err
		}
	}

	counts := dao.BoardFollowerCounts([]uint{board.ID})
	return follow, counts[board.ID], nil
}
