package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
)

type CommandExec struct {
	Id        int64               `json:"id"`
	Type      globals.CommandType `json:"type"`
	By        int64               `json:"by"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

func GetLatestSuccessCommandExecByType(typ globals.CommandType) (*CommandExec, error) {
	var result CommandExec

	err := globals.Pool.QueryRow("SELECT id, `type`, `by`, `status`, created_at, updated_at FROM command_exec WHERE type = ? AND status = 'success' ORDER BY updated_at DESC LIMIT 1", typ).
		Scan(&result.Id, &result.Type, &result.By, &result.Status, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &result, nil
}
