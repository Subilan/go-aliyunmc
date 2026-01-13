package store

import (
	"log"

	"github.com/Subilan/go-aliyunmc/events"
	"github.com/Subilan/go-aliyunmc/helpers/db"
)

func GetEventsOfTask(taskId string) ([]events.Event, error) {
	var result = make([]events.Event, 0)

	rows, err := db.Pool.Query("SELECT task_id, ord, type, is_error, is_public, content FROM pushed_events WHERE task_id = ? ORDER BY ord", taskId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var event events.Event
		err = rows.Scan(&event.TaskId, &event.Ord, &event.Type, &event.IsError, &event.IsPublic, &event.Content)

		if err != nil {
			log.Println("cannot scan row: ", err.Error())
			return nil, err
		}

		result = append(result, event)
	}

	return result, nil
}
