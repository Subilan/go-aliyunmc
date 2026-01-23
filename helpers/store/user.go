package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/consts"
)

type User struct {
	Id        int64           `json:"id"`
	Username  string          `json:"username"`
	CreatedAt time.Time       `json:"createdAt"`
	Role      consts.UserRole `json:"role"`
}
