package forummodels

import (
	"time"

	"gorm.io/gorm"
)

// Reply 是主题下的跟帖。floor 是稳定楼层号，删除后不重排。
type Reply struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"not null;index:idx_forum_replies_author_created,priority:2" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	ThreadID uint `gorm:"not null;uniqueIndex:idx_forum_replies_thread_floor,priority:1" json:"threadId"`
	AuthorID uint `gorm:"not null;index:idx_forum_replies_author_created,priority:1" json:"authorId"`
	// Floor 从 1 开始，删除后保留占位。
	Floor       uint   `gorm:"type:int unsigned;not null;uniqueIndex:idx_forum_replies_thread_floor,priority:2" json:"floor"`
	ContentHTML string `gorm:"type:longtext;not null" json:"contentHtml"`
	ContentText string `gorm:"type:mediumtext;not null" json:"contentText"`
	// ReplyToFloor 是正文中第一个有效引用的楼层。
	ReplyToFloor *uint `gorm:"type:int unsigned" json:"replyToFloor"`
	// ReplyToReplyID 是解析后的目标回复。
	ReplyToReplyID *uint `gorm:"index:idx_forum_replies_reply_to_reply_id" json:"replyToReplyId"`
	DeletedBy      *uint `json:"deletedBy"`
}

func (Reply) TableName() string { return "forum_replies" }

// Tag 是标签，normalized 为小写归一化后的唯一键。
type Tag struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"size:32;not null" json:"name"`
	Normalized string    `gorm:"size:32;not null;uniqueIndex:idx_forum_tags_normalized" json:"normalized"`
	CreatedAt  time.Time `gorm:"not null" json:"createdAt"`
}

func (Tag) TableName() string { return "forum_tags" }

// ThreadTag 是主题与标签的关联。
type ThreadTag struct {
	ThreadID  uint      `gorm:"primaryKey;autoIncrement:false;not null" json:"threadId"`
	TagID     uint      `gorm:"primaryKey;autoIncrement:false;not null;index:idx_forum_thread_tags_tag_id" json:"tagId"`
	CreatedAt time.Time `gorm:"not null" json:"createdAt"`
}

func (ThreadTag) TableName() string { return "forum_thread_tags" }
