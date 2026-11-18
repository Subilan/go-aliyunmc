// Package perms 定义论坛的权限点目录与论坛自持的 Casbin 权限判断。
// 权限点直接授予用户，不存在角色实体；超级用户拥有全部权限。
package perms

// Scope 是权限点可以授予的范围。
type Scope string

const (
	// ScopeGlobal 表示权限点只能按全局范围判断。
	ScopeGlobal Scope = "global"
	// ScopeBoard 表示权限点可以按板块授予，也可以全局授予。
	ScopeBoard Scope = "board"
	// ScopeSelf 是普通自助权限，默认授予新用户。
	ScopeSelf Scope = "self"
)

// Permission 是权限点目录中的一项。
type Permission struct {
	Code        string
	Group       string
	Scope       Scope
	Description string
	// Default 表示新用户首次登录时自动授予。
	Default bool
}

// catalog 是 v1 的权限点全集，必须与 moon-plan.md 5.2 保持一致。
var catalog = []Permission{
	{Code: "thread.create", Group: "内容", Scope: ScopeSelf, Description: "发主题", Default: true},
	{Code: "thread.update_own", Group: "内容", Scope: ScopeSelf, Description: "编辑自己的主题", Default: true},
	{Code: "thread.delete_own", Group: "内容", Scope: ScopeSelf, Description: "删除自己的主题", Default: true},
	{Code: "reply.create", Group: "内容", Scope: ScopeSelf, Description: "发跟帖", Default: true},
	{Code: "reply.update_own", Group: "内容", Scope: ScopeSelf, Description: "编辑自己的跟帖", Default: true},
	{Code: "reply.delete_own", Group: "内容", Scope: ScopeSelf, Description: "删除自己的跟帖", Default: true},
	{Code: "like.create", Group: "内容", Scope: ScopeSelf, Description: "点赞主题/回复/用户", Default: true},
	{Code: "board.follow", Group: "内容", Scope: ScopeSelf, Description: "关注板块", Default: true},
	{Code: "profile.update_own", Group: "用户", Scope: ScopeSelf, Description: "修改自己的资料/头像/背景", Default: true},
	{Code: "notification.read_own", Group: "用户", Scope: ScopeSelf, Description: "读取/标记自己的通知", Default: true},
	{Code: "upload.create", Group: "内容", Scope: ScopeSelf, Description: "上传图片", Default: true},

	{Code: "thread.pin", Group: "审核", Scope: ScopeBoard, Description: "置顶/取消置顶"},
	{Code: "thread.lock", Group: "审核", Scope: ScopeBoard, Description: "锁定/解锁"},
	{Code: "thread.move", Group: "审核", Scope: ScopeBoard, Description: "移动主题到其他板块"},
	{Code: "thread.update_any", Group: "审核", Scope: ScopeBoard, Description: "编辑任意主题"},
	{Code: "thread.delete_any", Group: "审核", Scope: ScopeBoard, Description: "删除任意主题"},
	{Code: "reply.update_any", Group: "审核", Scope: ScopeBoard, Description: "编辑任意跟帖"},
	{Code: "reply.delete_any", Group: "审核", Scope: ScopeBoard, Description: "删除任意跟帖"},
	{Code: "content.restore", Group: "审核", Scope: ScopeBoard, Description: "恢复已删除内容"},
	{Code: "content.view_admin", Group: "审核", Scope: ScopeBoard, Description: "访问后台内容列表"},

	{Code: "board.create", Group: "板块", Scope: ScopeGlobal, Description: "创建板块"},
	{Code: "board.update", Group: "板块", Scope: ScopeBoard, Description: "修改板块设置"},
	{Code: "board.delete", Group: "板块", Scope: ScopeGlobal, Description: "删除板块"},
	{Code: "board.reorder", Group: "板块", Scope: ScopeGlobal, Description: "板块排序"},

	{Code: "user.list", Group: "用户", Scope: ScopeGlobal, Description: "查看用户列表/详情"},
	{Code: "user.manage", Group: "用户", Scope: ScopeGlobal, Description: "修改用户资料"},
	{Code: "user.ban", Group: "用户", Scope: ScopeGlobal, Description: "封禁/解封"},

	{Code: "permission.manage", Group: "权限", Scope: ScopeGlobal, Description: "授予/撤销用户的权限节点"},
	{Code: "permission.view", Group: "权限", Scope: ScopeGlobal, Description: "查看权限点目录与用户授权"},

	{Code: "settings.manage", Group: "运营", Scope: ScopeGlobal, Description: "站点设置"},
	{Code: "upload.manage", Group: "运营", Scope: ScopeGlobal, Description: "上传文件管理"},
	{Code: "stats.view", Group: "运营", Scope: ScopeGlobal, Description: "数据看板"},
	{Code: "audit.view", Group: "运营", Scope: ScopeGlobal, Description: "审计日志"},

	{Code: "admin.access", Group: "后台", Scope: ScopeBoard, Description: "登录管理后台"},
}

var (
	byCode       = make(map[string]Permission, len(catalog))
	defaultCodes = make([]string, 0, 11)
	allCodes     = make([]string, 0, len(catalog))
)

func init() {
	for _, p := range catalog {
		byCode[p.Code] = p
		allCodes = append(allCodes, p.Code)
		if p.Default {
			defaultCodes = append(defaultCodes, p.Code)
		}
	}
}

// All 返回权限点目录的副本。
func All() []Permission {
	return append([]Permission(nil), catalog...)
}

// AllCodes 返回全部权限点 code。
func AllCodes() []string {
	return append([]string(nil), allCodes...)
}

// Get 按 code 查找权限点。
func Get(code string) (Permission, bool) {
	p, ok := byCode[code]
	return p, ok
}

// Exists 判断权限点是否仍在代码目录中。
func Exists(code string) bool {
	_, ok := byCode[code]
	return ok
}

// DefaultCodes 返回新用户首次登录时自动授予的权限点。
func DefaultCodes() []string {
	return append([]string(nil), defaultCodes...)
}
