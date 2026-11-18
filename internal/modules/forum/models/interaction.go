package forummodels

import "time"

// ThreadLike 是主题点赞关系。
type ThreadLike struct {
	ThreadID  uint      `gorm:"primaryKey;autoIncrement:false;not null" json:"threadId"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;not null;index:idx_forum_thread_likes_user_id" json:"userId"`
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
}

func (ThreadLike) TableName() string { return "forum_thread_likes" }

// ReplyLike 是跟帖点赞关系。
type ReplyLike struct {
	ReplyID   uint      `gorm:"primaryKey;autoIncrement:false;not null" json:"replyId"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;not null;index:idx_forum_reply_likes_user_id" json:"userId"`
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
}

func (ReplyLike) TableName() string { return "forum_reply_likes" }

// UserLike 是用户主页点赞关系。
type UserLike struct {
	TargetUserID uint      `gorm:"primaryKey;autoIncrement:false;not null" json:"targetUserId"`
	LikerID      uint      `gorm:"primaryKey;autoIncrement:false;not null;index:idx_forum_user_likes_liker_id" json:"likerId"`
	CreatedAt    time.Time `gorm:"not null" json:"createdAt"`
}

func (UserLike) TableName() string { return "forum_user_likes" }

// BoardFollow 是板块关注关系。
type BoardFollow struct {
	BoardID   uint      `gorm:"primaryKey;autoIncrement:false;not null" json:"boardId"`
	UserID    uint      `gorm:"primaryKey;autoIncrement:false;not null;index:idx_forum_board_follows_user_id" json:"userId"`
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
}

func (BoardFollow) TableName() string { return "forum_board_follows" }
