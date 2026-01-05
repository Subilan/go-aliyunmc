package store

import (
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
)

type CommandExec struct {
	Id        int64               `json:"id"`
	Type      globals.CommandType `json:"type"`
	By        *int64              `json:"by"`
	Status    string              `json:"status"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Auto      bool                `json:"auto"`
}

const q = "SELECT id, `type`, `by`, `status`, created_at, updated_at, auto FROM command_exec "

func GetLatestSuccessCommandExecByType(typ globals.CommandType) (*CommandExec, error) {
	var result CommandExec

	err := globals.Pool.QueryRow(q+"WHERE type = ? AND status = 'success' ORDER BY updated_at DESC LIMIT 1", typ).
		Scan(&result.Id, &result.Type, &result.By, &result.Status, &result.CreatedAt, &result.UpdatedAt, &result.Auto)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func GetExistingBackups() ([]*CommandExec, error) {
	var result = make([]*CommandExec, 0, 5)

	rows, err := globals.Pool.Query(q+"WHERE type = ? AND status = 'success' ORDER BY updated_at DESC LIMIT 5", globals.CmdTypeBackupWorlds)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var res CommandExec
		err = rows.Scan(&res.Id, &res.Type, &res.By, &res.Status, &res.CreatedAt, &res.UpdatedAt, &res.Auto)

		if err != nil {
			return nil, err
		}

		result = append(result, &res)
	}

	return result, nil
}
