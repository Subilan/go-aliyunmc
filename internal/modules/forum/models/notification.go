package forummodels

import "time"

// NotificationType 是通知类型。
type NotificationType string

const (
	NotificationTypeReply       NotificationType = "reply"
	NotificationTypeQuote       NotificationType = "quote"
	NotificationTypeLikeThread  NotificationType = "like_thread"
	NotificationTypeLikeReply   NotificationType = "like_reply"
	NotificationTypeLikeUser    NotificationType = "like_user"
	NotificationTypeMention     NotificationType = "mention"
	NotificationTypeBoardThread NotificationType = "board_thread"
)

// Notification 是持久化的站内通知。
type Notification struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// RecipientID 同时参与两个列表索引与去重唯一索引。
	RecipientID uint   `gorm:"not null;index:idx_forum_notifications_recipient_created,priority:1;index:idx_forum_notifications_recipient_read_created,priority:1;uniqueIndex:idx_forum_notifications_unique,priority:2" json:"recipientId"`
	ActorID     uint   `gorm:"not null;index:idx_forum_notifications_actor_id;uniqueIndex:idx_forum_notifications_unique,priority:3" json:"actorId"`
	Type        string `gorm:"size:16;not null;uniqueIndex:idx_forum_notifications_unique,priority:1" json:"type"`
	ThreadID    uint   `gorm:"not null;index:idx_forum_notifications_thread_id;uniqueIndex:idx_forum_notifications_unique,priority:4" json:"threadId"`
	ReplyID     uint   `gorm:"not null;default:0;index:idx_forum_notifications_reply_id;uniqueIndex:idx_forum_notifications_unique,priority:5" json:"replyId"`
	// QuotedFloor 是 quote 通知引用的楼层号；其他类型为 0。
	QuotedFloor uint       `gorm:"type:int unsigned;not null;default:0" json:"quotedFloor"`
	Excerpt     string     `gorm:"size:500;not null;default:''" json:"excerpt"`
	ReadAt      *time.Time `gorm:"type:datetime(3);index:idx_forum_notifications_recipient_read_created,priority:2" json:"readAt"`
	CreatedAt   time.Time  `gorm:"not null;index:idx_forum_notifications_recipient_created,priority:2;index:idx_forum_notifications_recipient_read_created,priority:3;index:idx_forum_notifications_created_at" json:"createdAt"`
}

func (Notification) TableName() string { return "forum_notifications" }
