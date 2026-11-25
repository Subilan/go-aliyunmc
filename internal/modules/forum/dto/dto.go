// Package dto 定义论坛 API 的稳定 JSON 契约。
package dto

import (
	"encoding/json"

	"github.com/Subilan/go-aliyunmc/internal/modules/forum/perms"
)

// Page 是统一分页响应。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// NewPage 构造分页响应，保证空列表返回 [] 而不是 null。
func NewPage[T any](items []T, total int64, page, pageSize int) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Total: total, Page: page, PageSize: pageSize}
}

// UserSummary 是嵌入列表/卡片的最小用户信息。
type UserSummary struct {
	ID          uint    `json:"id"`
	Username    string  `json:"username"`
	Nickname    *string `json:"nickname"`
	AvatarURL   string  `json:"avatarUrl"`
	AvatarShape string  `json:"avatarShape"`
}

// UserProfile 是用户主页资料。
type UserProfile struct {
	ID               uint    `json:"id"`
	Username         string  `json:"username"`
	Nickname         *string `json:"nickname"`
	Bio              string  `json:"bio"`
	AvatarURL        string  `json:"avatarUrl"`
	BackgroundURL    *string `json:"backgroundUrl"`
	AvatarShape      string  `json:"avatarShape"`
	JoinedAt         string  `json:"joinedAt"`
	ThreadCount      int64   `json:"threadCount"`
	ReplyCount       int64   `json:"replyCount"`
	LikeCount        int64   `json:"likeCount"`
	ProfileLikeCount int64   `json:"profileLikeCount"`
	LikedByMe        bool    `json:"likedByMe"`
	IsSelf           bool    `json:"isSelf"`
}

// BoardBrief 是嵌入主题卡片的板块信息。
type BoardBrief struct {
	ID    uint   `json:"id"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// BoardSummary 是板块列表项。
type BoardSummary struct {
	ID            uint   `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Icon          string `json:"icon"`
	Color         string `json:"color"`
	ThreadCount   int64  `json:"threadCount"`
	FollowerCount int64  `json:"followerCount"`
	FollowingByMe bool   `json:"followingByMe"`
}

// ThreadSummary 是主题列表卡片。
type ThreadSummary struct {
	ID         uint         `json:"id"`
	Board      BoardBrief   `json:"board"`
	Author     *UserSummary `json:"author"`
	Title      string       `json:"title"`
	Excerpt    string       `json:"excerpt"`
	Tags       []string     `json:"tags"`
	Pinned     bool         `json:"pinned"`
	Locked     bool         `json:"locked"`
	ReplyCount int64        `json:"replyCount"`
	LikeCount  int64        `json:"likeCount"`
	LikedByMe  bool         `json:"likedByMe"`
	CreatedAt  string       `json:"createdAt"`
	UpdatedAt  string       `json:"updatedAt"`
}

// ThreadDetail 是主题详情。匿名嵌入使 JSON 字段与 ThreadSummary 一致。
type ThreadDetail struct {
	ThreadSummary
	ContentHTML string `json:"contentHtml"`
	CanEdit     bool   `json:"canEdit"`
	CanDelete   bool   `json:"canDelete"`
	CanModerate bool   `json:"canModerate"`
}

// Reply 是跟帖。删除后作为占位返回，作者与正文为 null。
// ThreadTitle 仅在用户主页等跨主题列表中使用，主题内跟帖列表不返回。
type Reply struct {
	ID             uint         `json:"id"`
	ThreadID       uint         `json:"threadId"`
	ThreadTitle    string       `json:"threadTitle,omitempty"`
	Floor          uint         `json:"floor"`
	Author         *UserSummary `json:"author"`
	ContentHTML    *string      `json:"contentHtml"`
	ReplyToFloor   *uint        `json:"replyToFloor"`
	ReplyToReplyID *uint        `json:"replyToReplyId"`
	LikeCount      int64        `json:"likeCount"`
	LikedByMe      bool         `json:"likedByMe"`
	Deleted        bool         `json:"deleted"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      *string      `json:"updatedAt"`
	CanEdit        bool         `json:"canEdit"`
	CanDelete      bool         `json:"canDelete"`
}

// Notification 是站内通知。
type Notification struct {
	ID          uint         `json:"id"`
	Type        string       `json:"type"`
	Actor       *UserSummary `json:"actor"`
	ThreadID    uint         `json:"threadId"`
	ThreadTitle string       `json:"threadTitle"`
	ReplyID     uint         `json:"replyId"`
	QuotedFloor uint         `json:"quotedFloor"`
	Excerpt     string       `json:"excerpt"`
	Read        bool         `json:"read"`
	CreatedAt   string       `json:"createdAt"`
}

// RecentReply 是最近回复侧栏项。
type RecentReply struct {
	ID          uint         `json:"id"`
	ThreadID    uint         `json:"threadId"`
	ThreadTitle string       `json:"threadTitle"`
	BoardSlug   string       `json:"boardSlug"`
	Floor       uint         `json:"floor"`
	Excerpt     string       `json:"excerpt"`
	Author      *UserSummary `json:"author"`
	CreatedAt   string       `json:"createdAt"`
}

// TagSummary 是热门标签项。
type TagSummary struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// Site 是公开站点信息。
type Site struct {
	SiteName        string `json:"siteName"`
	SiteDescription string `json:"siteDescription"`
	Announcement    string `json:"announcement"`
	FooterText      string `json:"footerText"`
	AllowGuestRead  bool   `json:"allowGuestRead"`
}

// Me 是登录态与权限响应。
type Me struct {
	User              *UserSummary        `json:"user"`
	GlobalPermissions []string            `json:"globalPermissions"`
	BoardPermissions  map[string][]string `json:"boardPermissions"`
}

// AdminMe 是后台登录态与权限响应。
type AdminMe struct {
	User              *UserSummary        `json:"user"`
	IsSuperuser       bool                `json:"isSuperuser"`
	GlobalPermissions []string            `json:"globalPermissions"`
	BoardPermissions  map[string][]string `json:"boardPermissions"`
}

// LoginResult 是登录成功响应。
type LoginResult struct {
	User              UserSummary         `json:"user"`
	GlobalPermissions []string            `json:"globalPermissions"`
	BoardPermissions  map[string][]string `json:"boardPermissions"`
}

// UploadResult 是上传成功响应。
type UploadResult struct {
	ID     uint   `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Mime   string `json:"mime"`
	Size   int64  `json:"size"`
}

// PermissionItem 是权限点目录项。
type PermissionItem struct {
	Code        string `json:"code"`
	Group       string `json:"group"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// LikeResult 是点赞/关注类接口的返回。
type LikeResult struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"likeCount"`
}

// AdminUser 是后台用户列表/详情项。
type AdminUser struct {
	ID             uint          `json:"id"`
	Username       string        `json:"username"`
	Nickname       *string       `json:"nickname"`
	Bio            string        `json:"bio"`
	AvatarURL      string        `json:"avatarUrl"`
	AvatarShape    string        `json:"avatarShape"`
	Banned         bool          `json:"banned"`
	LabRole        string        `json:"labRole"`
	CreatedAt      string        `json:"createdAt"`
	LastForumLogin *string       `json:"lastForumLoginAt"`
	LastAdminLogin *string       `json:"lastAdminLoginAt"`
	ThreadCount    int64         `json:"threadCount"`
	ReplyCount     int64         `json:"replyCount"`
	LikeCount      int64         `json:"likeCount"`
	Permissions    []perms.Grant `json:"permissions"`
}

// AdminBoard 是后台板块列表项。
type AdminBoard struct {
	ID          uint   `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sortOrder"`
	Enabled     bool   `json:"enabled"`
	Deleted     bool   `json:"deleted"`
	ThreadCount int64  `json:"threadCount"`
}

// AdminThread 是后台主题列表/详情项。
type AdminThread struct {
	ID          uint         `json:"id"`
	BoardID     uint         `json:"boardId"`
	Board       BoardBrief   `json:"board"`
	Author      *UserSummary `json:"author"`
	Title       string       `json:"title"`
	ContentHTML string       `json:"contentHtml,omitempty"`
	Excerpt     string       `json:"excerpt"`
	Pinned      bool         `json:"pinned"`
	Locked      bool         `json:"locked"`
	Deleted     bool         `json:"deleted"`
	DeletedAt   *string      `json:"deletedAt"`
	DeletedBy   *uint        `json:"deletedBy"`
	ReplyCount  int64        `json:"replyCount"`
	LikeCount   int64        `json:"likeCount"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
	LastReplyAt *string      `json:"lastReplyAt"`
}

// AdminReply 是后台跟帖列表项。
type AdminReply struct {
	ID          uint         `json:"id"`
	ThreadID    uint         `json:"threadId"`
	ThreadTitle string       `json:"threadTitle"`
	Floor       uint         `json:"floor"`
	Author      *UserSummary `json:"author"`
	ContentHTML string       `json:"contentHtml,omitempty"`
	Excerpt     string       `json:"excerpt"`
	Deleted     bool         `json:"deleted"`
	DeletedAt   *string      `json:"deletedAt"`
	DeletedBy   *uint        `json:"deletedBy"`
	LikeCount   int64        `json:"likeCount"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
}

// AdminUpload 是后台上传列表项。
type AdminUpload struct {
	ID           uint   `json:"id"`
	OwnerID      uint   `json:"ownerId"`
	OwnerName    string `json:"ownerName"`
	Kind         string `json:"kind"`
	Key          string `json:"key"`
	URL          string `json:"url"`
	OriginalName string `json:"originalName"`
	Mime         string `json:"mime"`
	SizeBytes    int64  `json:"sizeBytes"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	SHA256       string `json:"sha256"`
	CreatedAt    string `json:"createdAt"`
}

// UploadStats 是上传分类统计。
type UploadStats struct {
	Kind      string `json:"kind"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"sizeBytes"`
}

// UploadStatsSummary 是上传分类统计与磁盘配额。
// QuotaBytes / UserQuotaBytes 为 0 表示不限制。
type UploadStatsSummary struct {
	Items          []UploadStats `json:"items"`
	QuotaBytes     int64         `json:"quotaBytes"`
	UserQuotaBytes int64         `json:"userQuotaBytes"`
}

// DashboardTotals 是看板总量。
type DashboardTotals struct {
	Threads int64 `json:"threads"`
	Replies int64 `json:"replies"`
	Users   int64 `json:"users"`
	Boards  int64 `json:"boards"`
}

// DashboardTrend 是看板按天趋势。
type DashboardTrend struct {
	Date    string `json:"date"`
	Threads int64  `json:"threads"`
	Replies int64  `json:"replies"`
	Users   int64  `json:"users"`
}

// Dashboard 是后台看板响应。
type Dashboard struct {
	Totals DashboardTotals  `json:"totals"`
	Today  DashboardTotals  `json:"today"`
	Trend  []DashboardTrend `json:"trend"`
}

// SiteSettings 是后台设置响应。
type SiteSettings struct {
	SiteName                 string `json:"siteName"`
	SiteDescription          string `json:"siteDescription"`
	Announcement             string `json:"announcement"`
	FooterText               string `json:"footerText"`
	AllowGuestRead           bool   `json:"allowGuestRead"`
	DefaultAvatarURLTemplate string `json:"defaultAvatarUrlTemplate"`
	RegistrationEnabled      bool   `json:"registrationEnabled"`
	// Raw 保留数据库键值形式，便于后台按 key 保存。
	Raw map[string]string `json:"raw"`
}

// AuditLog 是后台审计日志项。
type AuditLog struct {
	ID         uint            `json:"id"`
	ActorID    uint            `json:"actorId"`
	Action     string          `json:"action"`
	TargetType string          `json:"targetType"`
	TargetID   uint            `json:"targetId"`
	Detail     json.RawMessage `json:"detail"`
	IP         string          `json:"ip"`
	UserAgent  string          `json:"userAgent"`
	CreatedAt  string          `json:"createdAt"`
}

// LoginRecordItem 是后台登录记录项。
type LoginRecordItem struct {
	ID        uint   `json:"id"`
	Channel   string `json:"channel"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
	CreatedAt string `json:"createdAt"`
}

// BatchFailure 是批量操作中单个失败项。
type BatchFailure struct {
	ID     uint   `json:"id"`
	Reason string `json:"reason"`
}

// BatchResult 是批量操作结果。
type BatchResult struct {
	Succeeded []uint         `json:"succeeded"`
	Failed    []BatchFailure `json:"failed"`
}
