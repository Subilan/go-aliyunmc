package service

import (
	"github.com/Subilan/go-aliyunmc/internal/common/sse"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dto"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/realtime"
)

type notificationPayload struct {
	Notification dto.Notification `json:"notification"`
	UnreadCount  int64            `json:"unreadCount"`
}

type notificationReadPayload struct {
	UnreadCount int64  `json:"unreadCount"`
	IDs         []uint `json:"ids,omitempty"`
	All         bool   `json:"all,omitempty"`
}

type threadLikePayload struct {
	ThreadID  uint  `json:"threadId"`
	LikeCount int64 `json:"likeCount"`
	ActorID   uint  `json:"actorId"`
	Liked     bool  `json:"liked"`
}

type replyLikePayload struct {
	ThreadID  uint  `json:"threadId"`
	ReplyID   uint  `json:"replyId"`
	LikeCount int64 `json:"likeCount"`
	ActorID   uint  `json:"actorId"`
	Liked     bool  `json:"liked"`
}

type userLikePayload struct {
	UserID           uint  `json:"userId"`
	ProfileLikeCount int64 `json:"profileLikeCount"`
	ActorID          uint  `json:"actorId"`
	Liked            bool  `json:"liked"`
}

type boardFollowPayload struct {
	BoardID       uint   `json:"boardId"`
	BoardSlug     string `json:"boardSlug"`
	FollowerCount int64  `json:"followerCount"`
	ActorID       uint   `json:"actorId"`
	Following     bool   `json:"following"`
}

// publishNotifications 只向有活跃 SSE 连接的通知接收者推送。
// 事务提交后调用；构造 DTO 需要的 actor 与主题信息由批量查询共享。
func publishNotifications(rows []forummodels.Notification) {
	if len(rows) == 0 || realtime.Current() == nil {
		return
	}

	active := make([]forummodels.Notification, 0, len(rows))
	for _, row := range rows {
		if row.RecipientID != 0 && realtime.HasSubscriber(row.RecipientID) {
			active = append(active, row)
		}
	}
	if len(active) == 0 {
		return
	}

	actorIDs := make([]uint, 0, len(active))
	threadIDs := make([]uint, 0, len(active))
	recipientIDs := make([]uint, 0, len(active))
	for _, row := range active {
		actorIDs = append(actorIDs, row.ActorID)
		threadIDs = append(threadIDs, row.ThreadID)
		recipientIDs = append(recipientIDs, row.RecipientID)
	}

	views := dao.LoadUserViews(uniqueIDs(actorIDs))
	titles := dao.ThreadTitles(uniqueIDs(threadIDs))
	items := dto.Notifications(active, views, titles)
	counts := dao.UnreadNotificationCounts(uniqueIDs(recipientIDs))

	for i, row := range active {
		realtime.PublishUser(row.RecipientID, sse.Event{
			Event: realtime.EventNotification,
			Data: notificationPayload{
				Notification: items[i],
				UnreadCount:  counts[row.RecipientID],
			},
		})
	}
}

// publishNotificationRead 让当前用户的其它标签页同步已读状态。
func publishNotificationRead(userID uint, unread int64, ids []uint, all bool) {
	realtime.PublishUser(userID, sse.Event{
		Event: realtime.EventNotificationRead,
		Data: notificationReadPayload{
			UnreadCount: unread,
			IDs:         ids,
			All:         all,
		},
	})
}

func publishThreadLike(threadID uint, likeCount int64, actorID uint, liked bool) {
	realtime.PublishAll(sse.Event{
		Event: realtime.EventThreadLike,
		Data: threadLikePayload{
			ThreadID:  threadID,
			LikeCount: likeCount,
			ActorID:   actorID,
			Liked:     liked,
		},
	})
}

func publishReplyLike(threadID, replyID uint, likeCount int64, actorID uint, liked bool) {
	realtime.PublishAll(sse.Event{
		Event: realtime.EventReplyLike,
		Data: replyLikePayload{
			ThreadID:  threadID,
			ReplyID:   replyID,
			LikeCount: likeCount,
			ActorID:   actorID,
			Liked:     liked,
		},
	})
}

func publishUserLike(userID uint, profileLikeCount int64, actorID uint, liked bool) {
	realtime.PublishAll(sse.Event{
		Event: realtime.EventUserLike,
		Data: userLikePayload{
			UserID:           userID,
			ProfileLikeCount: profileLikeCount,
			ActorID:          actorID,
			Liked:            liked,
		},
	})
}

func publishBoardFollow(boardID uint, slug string, followerCount int64, actorID uint, following bool) {
	realtime.PublishAll(sse.Event{
		Event: realtime.EventBoardFollow,
		Data: boardFollowPayload{
			BoardID:       boardID,
			BoardSlug:     slug,
			FollowerCount: followerCount,
			ActorID:       actorID,
			Following:     following,
		},
	})
}
