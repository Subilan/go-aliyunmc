package perms

import "github.com/gin-gonic/gin"

// Role 表示一个权限等级。权限等级之间是有序的，其权限等级与数值之间的关系通过levelMap映射，数值越大表示权限越高。
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

// Normal 返回一个 Role 的规范化字符串表示，用于与policy文件中的role（subject）进行匹配。
func (r Role) Normal() string {
	if r == RoleBasic {
		return "basic"
	}
	return string(r)
}

// CanRequest 判断该权限等级是否可以以 c.Request.Method 方法访问 c.Request.URL.Path 路径
func (r Role) CanRequest(c *gin.Context) (bool, error) {
	obj := c.Request.URL.Path
	act := c.Request.Method

	return Enforce(r, obj, act)
}

// CanExecute 判断该权限等级是否可以对 target 进行 execute 操作
func (r Role) CanExecute(target string) (bool, error) {
	obj := target
	act := "execute"

	return Enforce(r, obj, act)
}

// Gt 判断该权限等级是否严格高于另一个权限等级
func (r Role) Gt(other Role) bool {
	return levelMap[r] > levelMap[other]
}

// Gte 判断该权限等级是否高于或相当于另一个权限等级
func (r Role) Gte(other Role) bool {
	return levelMap[r] >= levelMap[other]
}
