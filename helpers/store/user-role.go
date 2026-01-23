package store

import (
	"database/sql"
	"errors"

	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

func GetUserRole(userId int64, fallbackRole consts.UserRole) (consts.UserRole, error) {
	var role consts.UserRole

	err := db.Pool.QueryRow("SELECT `role` FROM user_roles WHERE user_id=?", userId).Scan(&role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fallbackRole, nil
		}
		return consts.UserRoleEmpty, err
	}

	return role, nil
}
