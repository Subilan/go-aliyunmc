package perm_util

// Role 表示一个权限等级。权限等级之间是有序的，其权限等级与数值之间的关系通过 levelMap 映射，数值越大表示权限越高。
type Role string

const (
	// RoleBasic 是最低的权限等级，表示任意登录的用户
	RoleBasic Role = ""
	// RoleOperator 表示操作员权限等级
	RoleOperator Role = "operator"
	// RoleSuperuser 表示超级用户权限等级
	RoleSuperuser Role = "superuser"
)

var levelMap = map[Role]int{
	RoleBasic:     1,
	RoleOperator:  2,
	RoleSuperuser: 3,
}

// Normal 返回一个 Role 的规范化字符串表示，用于与 policy 文件中的 role（subject）进行匹配。
func (r Role) Normal() string {
	if r == RoleBasic {
		return "basic"
	}
	return string(r)
}

// IsValid 判断角色是否为系统支持的角色。
func (r Role) IsValid() bool {
	_, ok := levelMap[r]
	return ok
}

// Gt 判断该权限等级是否严格高于另一个权限等级
func (r Role) Gt(other Role) bool {
	return levelMap[r] > levelMap[other]
}

// Gte 判断该权限等级是否高于或相当于另一个权限等级
func (r Role) Gte(other Role) bool {
	return levelMap[r] >= levelMap[other]
}
