package consts

// UserRole 表示用户的权限角色
type UserRole int

const (

	// UserRoleEmpty 表示空的权限角色，也可以理解为默认权限角色。默认权限角色在不同版本的代码中可能有不同的处理方式，一般认为其 fallback 到 UserRoleUser 上
	UserRoleEmpty UserRole = iota
	// UserRoleUser 表示普通用户权限角色
	UserRoleUser
	// UserRoleAdmin 表示超级用户权限角色
	UserRoleAdmin
)
