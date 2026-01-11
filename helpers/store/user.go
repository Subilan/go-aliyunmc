package store

import "time"

type User struct {
	Id        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	Role      string    `json:"role"`
}
