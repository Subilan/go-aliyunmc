package tasks

import (
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CreateTask 初始化一个新的任务并立即写入到数据库。by 是创建任务的用户ID。
func CreateTask(taskType models.TaskType, by *uint) (*models.Task, error) {
	task := &models.Task{
		Type:   taskType,
		Status: models.TaskStatusCreated,
		Output: "",
		Error:  "",
		Step:   0,
		By:     by,
	}
	result := store.DB.Create(task)
	return task, result.Error
}

// UpdateTask 将任务数据写入到数据库。task 是要更新的任务结构体指针。
func UpdateTask(task *models.Task) error {
	result := store.DB.Session(&gorm.Session{
		Logger: store.DB.Logger.LogMode(logger.Silent),
	}).Save(task)
	return result.Error
}

// ListTasks 获取指定状态的任务列表。若不指定状态，则返回任意状态的任务。
func ListTasks(status *models.TaskStatus, limit, offset int) ([]models.Task, error) {
	var tasks []models.Task
	query := store.DB.Model(&models.Task{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	result := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks)
	return tasks, result.Error
}
