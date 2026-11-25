package config

// Forum 对应 [forum]：论坛模块的配置。
type Forum struct {
	// UploadDir 是上传文件根目录；留空时使用 $STATE_DIR/state/forum/uploads。
	UploadDir string `toml:"upload_dir" comment:"上传文件根目录；留空表示 $STATE_DIR/state/forum/uploads"`

	// MaxUploadBytes 是正文图片上传的最大字节数。
	MaxUploadBytes int64 `toml:"max_upload_bytes" default:"10485760" comment:"正文图片上传的最大字节数"`

	// MaxAvatarBytes 是头像上传的最大字节数。
	MaxAvatarBytes int64 `toml:"max_avatar_bytes" default:"2097152" comment:"头像上传的最大字节数"`

	// MaxBackgroundBytes 是主页背景上传的最大字节数。
	MaxBackgroundBytes int64 `toml:"max_background_bytes" default:"8388608" comment:"主页背景上传的最大字节数"`

	// MaxThreadHTMLBytes 是主题正文 HTML 的最大字节数。
	MaxThreadHTMLBytes int `toml:"max_thread_html_bytes" default:"131072" comment:"主题正文 HTML 的最大字节数"`

	// MaxReplyHTMLBytes 是跟帖正文 HTML 的最大字节数。
	MaxReplyHTMLBytes int `toml:"max_reply_html_bytes" default:"65536" comment:"跟帖正文 HTML 的最大字节数"`

	// MaxImagePixels 是单张图片允许的最大总像素数。
	MaxImagePixels int `toml:"max_image_pixels" default:"24000000" comment:"单张图片允许的最大总像素数"`

	// MaxImagesPerPost 是单篇正文允许的最大图片数。
	MaxImagesPerPost int `toml:"max_images_per_post" default:"20" comment:"单篇正文允许的最大图片数"`

	// MaxMentionsPerPost 是单篇正文允许的最大提及数。
	MaxMentionsPerPost int `toml:"max_mentions_per_post" default:"20" comment:"单篇正文允许的最大提及数"`

	// MaxTagsPerPost 是单篇主题允许的最大标签数；0 表示不限制。
	MaxTagsPerPost int `toml:"max_tags_per_post" default:"20" comment:"单篇主题允许的最大标签数；0 表示不限制"`

	// MaxQuoteRefsPerPost 是单篇正文允许的最大楼层引用数；0 表示不限制。
	MaxQuoteRefsPerPost int `toml:"max_quote_refs_per_post" default:"20" comment:"单篇正文允许的最大楼层引用数；0 表示不限制"`

	// LikeNotificationCooldownMinutes 是点赞通知的抑制窗口：
	// 未读且在该窗口内重复点赞不再通知；已读或超窗则刷新同一条通知。0 表示不抑制。
	LikeNotificationCooldownMinutes int `toml:"like_notification_cooldown_minutes" default:"10" comment:"点赞通知抑制窗口（分钟）；0 表示不抑制"`

	// UserQuotaBytes 是每个用户上传文件占用的磁盘上限；0 表示不限制。
	UserQuotaBytes int64 `toml:"user_quota_bytes" default:"268435456" comment:"每个用户上传文件占用的磁盘上限（字节）；0 表示不限制"`

	// StorageQuotaBytes 是论坛上传文件允许占用的总磁盘上限；0 表示不限制。
	StorageQuotaBytes int64 `toml:"storage_quota_bytes" default:"1073741824" comment:"论坛上传文件总占用上限（字节）；0 表示不限制"`

	// RateLimit 是论坛接口的进程内限流配置。
	RateLimit ForumRateLimit `toml:"rate_limit"`
}

// ForumRateLimit 对应 [forum.rate_limit]。
type ForumRateLimit struct {
	// Enabled 控制是否启用限流。
	Enabled bool `toml:"enabled" default:"true" comment:"是否启用论坛接口限流"`

	LoginPerMinute         int `toml:"login_per_minute" default:"5" comment:"登录：每分钟每 IP+用户名的请求数"`
	LoginIPPerMinute       int `toml:"login_ip_per_minute" default:"30" comment:"登录：每分钟每 IP 的总请求数，防止跨用户名爆破"`
	ThreadPerMinute        int `toml:"thread_per_minute" default:"10" comment:"发主题：每分钟每用户的请求数"`
	EditPerMinute          int `toml:"edit_per_minute" default:"20" comment:"编辑主题/跟帖：每分钟每用户的请求数"`
	ReplyPerMinute         int `toml:"reply_per_minute" default:"20" comment:"发跟帖：每分钟每用户的请求数"`
	UploadPerMinute        int `toml:"upload_per_minute" default:"20" comment:"上传：每分钟每用户的请求数"`
	LikePerMinute          int `toml:"like_per_minute" default:"60" comment:"点赞/关注：每分钟每用户的请求数"`
	MentionSearchPerMinute int `toml:"mention_search_per_minute" default:"60" comment:"提及候选搜索：每分钟每用户的请求数"`
	SearchPerMinute        int `toml:"search_per_minute" default:"30" comment:"全文搜索：每分钟每用户（游客按 IP）的请求数"`
	AdminWritePerMinute    int `toml:"admin_write_per_minute" default:"60" comment:"后台写操作：每分钟每用户的请求数"`
}
