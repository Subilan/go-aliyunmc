package bot

import "github.com/gin-gonic/gin"

const principalKey = "bot_principal"

// Principal 是已通过签名认证的机器人身份。
type Principal struct {
	KeyID string
}

// SetPrincipal 把机器人身份写入 gin.Context。
func SetPrincipal(c *gin.Context, p *Principal) {
	c.Set(principalKey, p)
}

// GetPrincipal 从 gin.Context 读取机器人身份。
func GetPrincipal(c *gin.Context) (*Principal, bool) {
	value, exists := c.Get(principalKey)
	if !exists {
		return nil, false
	}

	p, ok := value.(*Principal)
	return p, ok
}
