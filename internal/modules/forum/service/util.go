package service

import (
	"net/http"

	"github.com/Subilan/go-aliyunmc/internal/common/h"
	"github.com/Subilan/go-aliyunmc/internal/modules/forum/dao"
	forummodels "github.com/Subilan/go-aliyunmc/internal/modules/forum/models"
)

// requireEnabledBoard 读取并校验板块存在且启用。
func requireEnabledBoard(boardID uint) (*forummodels.Board, error) {
	board, err := dao.BoardByID(boardID, false)
	if err != nil || !board.Enabled {
		return nil, h.HttpError(http.StatusNotFound, "板块不存在或已禁用")
	}
	return board, nil
}

// requireForumParticipant 把从未登录过论坛前台的共享账号视为不存在。
// 账号由 lab 创建，首次登录论坛后才成为论坛用户。
func requireForumParticipant(userID uint) error {
	ok, err := dao.HasForumLogin(userID)
	if err != nil {
		return err
	}
	if !ok {
		return h.HttpError(http.StatusNotFound, "用户不存在")
	}
	return nil
}

// uniqueIDs 按出现顺序返回去重后的 ID 列表。
func uniqueIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uint]bool, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}
