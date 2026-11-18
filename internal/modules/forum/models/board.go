package forummodels

import (
	"time"

	"gorm.io/gorm"
)

// Board 是论坛板块。slug 创建后不可修改，软删除后仍然占用。
type Board struct {
	gorm.Model
	Slug        string `gorm:"size:64;not null;uniqueIndex:idx_forum_boards_slug" json:"slug"`
	Name        string `gorm:"size:64;not null" json:"name"`
	Description string `gorm:"size:255;not null;default:''" json:"description"`
	Icon        string `gorm:"size:32;not null;default:'discussion'" json:"icon"`
	Color       string `gorm:"size:7;not null;default:'#6366f1'" json:"color"`
	SortOrder   int    `gorm:"not null;default:0;index:idx_forum_boards_enabled_sort_order,priority:2" json:"sortOrder"`
	Enabled     bool   `gorm:"not null;default:true;index:idx_forum_boards_enabled_sort_order,priority:1" json:"enabled"`
}

func (Board) TableName() string { return "forum_boards" }

// Thread 是论坛主题。软删除只影响可见性，不物理删除关联数据。
type Thread struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// CreatedAt 参与多个复合索引，因此不能直接复用 gorm.Model。
	CreatedAt time.Time      `gorm:"not null;index:idx_forum_threads_board_created,priority:2;index:idx_forum_threads_author_created,priority:2;index:idx_forum_threads_pinned_created,priority:2;index:idx_forum_threads_board_pinned_created,priority:3" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	BoardID  uint `gorm:"not null;index:idx_forum_threads_board_created,priority:1;index:idx_forum_threads_board_pinned_created,priority:1" json:"boardId"`
	AuthorID uint `gorm:"not null;index:idx_forum_threads_author_created,priority:1" json:"authorId"`
	// Title 是 1~120 rune 的纯文本标题。
	Title       string     `gorm:"size:160;not null" json:"title"`
	ContentHTML string     `gorm:"type:longtext;not null" json:"contentHtml"`
	ContentText string     `gorm:"type:mediumtext;not null" json:"contentText"`
	Pinned      bool       `gorm:"not null;default:false;index:idx_forum_threads_pinned_created,priority:1;index:idx_forum_threads_board_pinned_created,priority:2" json:"pinned"`
	PinnedAt    *time.Time `gorm:"type:datetime(3)" json:"pinnedAt"`
	Locked      bool       `gorm:"not null;default:false" json:"locked"`
	LastReplyAt *time.Time `gorm:"type:datetime(3);index:idx_forum_threads_last_reply_at" json:"lastReplyAt"`
	DeletedBy   *uint      `json:"deletedBy"`
}

func (Thread) TableName() string { return "forum_threads" }
