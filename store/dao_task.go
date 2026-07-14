package store

import (
	"github.com/Subilan/go-aliyunmc/store/models"
	"time"

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
	result := DB.Create(task)
	return task, result.Error
}

// UpdateTask 将任务数据写入到数据库。task 是要更新的任务结构体指针。
func UpdateTask(task *models.Task) error {
	result := DB.Session(&gorm.Session{
		Logger: DB.Logger.LogMode(logger.Silent),
	}).Save(task)
	return result.Error
}

var taskSortableFields = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"start_at":   true,
	"end_at":     true,
}

// ListTasks 获取指定状态的任务列表。若不指定状态，则返回任意状态的任务。
// sort 为排序字段，order 为排序方向（asc/desc）。若 sort 不在白名单内，则默认按 created_at DESC 排序。
func ListTasks(status *models.TaskStatus, sort, order string, limit, offset int) ([]models.Task, error) {
	var tasks []models.Task
	query := DB.Model(&models.Task{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if taskSortableFields[sort] && (order == "asc" || order == "desc") {
		query = query.Order(sort + " " + order)
	} else {
		query = query.Order("created_at DESC")
	}

	result := query.Preload("User").Omit("output").Limit(limit).Offset(offset).Find(&tasks)
	return tasks, result.Error
}

// CountTasks 返回任务总数，可按状态过滤。
func CountTasks(status *models.TaskStatus) (int64, error) {
	var count int64
	query := DB.Model(&models.Task{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	result := query.Count(&count)
	return count, result.Error
}

// TaskStats 包含任务概览统计数据。
type TaskStats struct {
	Total           int64      `json:"total"`
	SuccessCount    int64      `json:"successCount"`
	LastCompletedAt *time.Time `json:"lastCompletedAt"`
	LastCreatedBy   *uint      `json:"lastCreatedBy"`
	LastCreatedUser *models.User `json:"lastCreatedUser"`
}

// GetTaskStats 返回任务概览统计数据。
func GetTaskStats() (TaskStats, error) {
	var stats TaskStats

	if err := DB.Model(&models.Task{}).Count(&stats.Total).Error; err != nil {
		return stats, err
	}

	successStatus := models.TaskStatusSuccess
	if err := DB.Model(&models.Task{}).Where("status = ?", successStatus).Count(&stats.SuccessCount).Error; err != nil {
		return stats, err
	}

	var lastCompleted models.Task
	if err := DB.Where("status = ?", successStatus).Order("end_at DESC").First(&lastCompleted).Error; err == nil {
		stats.LastCompletedAt = lastCompleted.EndAt
	}

	var lastCreated models.Task
	if err := DB.Preload("User").Order("created_at DESC").First(&lastCreated).Error; err == nil {
		stats.LastCreatedBy = lastCreated.By
		stats.LastCreatedUser = lastCreated.User
	}

	return stats, nil
}

// GetTask 根据任务ID获取任务详情。
func GetTask(taskId uint) (*models.Task, error) {
	var task models.Task
	result := DB.Preload("User").First(&task, taskId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}
