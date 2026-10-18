package store

import (
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/store/models"

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

// UpdateTask 将任务的可变字段写入数据库。输出内容已拆分到 task_output_lines，
// 因此这里显式排除 output 列，避免每次落库都全量写入旧的大字段。
func UpdateTask(task *models.Task) error {
	result := DB.Session(&gorm.Session{
		Logger: DB.Logger.LogMode(logger.Silent),
	}).Model(task).
		Select("start_at", "end_at", "status", "step", "error").
		Updates(task)
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
	Total               int64                          `json:"total"`
	SuccessCount        int64                          `json:"successCount"`
	LastCompletedAt     *time.Time                     `json:"lastCompletedAt"`
	LastCreatedBy       *uint                          `json:"lastCreatedBy"`
	LastCreatedUser     *models.User                   `json:"lastCreatedUser"`
	TotalRuntimeSec     int64                          `json:"totalRuntimeSec"`
	LastCompletedByType map[models.TaskType]*time.Time `json:"lastCompletedByType"`
}

// GetTaskStats 返回任务概览统计数据。
func GetTaskStats() (TaskStats, error) {
	var stats TaskStats
	stats.LastCompletedByType = make(map[models.TaskType]*time.Time)

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

	// 计算总运行时长：按 启动服务器—archive 配对。
	// oneclick_start_game 成功意味着服务器已经启动，因此也作为启动标记参与配对。
	var pairedTasks []models.Task
	DB.Where("status = ? AND type IN ?", successStatus, []models.TaskType{
		models.TaskTypeStartServer,
		models.TaskTypeOneClickStartGame,
		models.TaskTypeArchive,
	}).Order("end_at ASC").Find(&pairedTasks)

	var totalRuntime time.Duration
	var lastStartEnd *time.Time
	for _, t := range pairedTasks {
		if (t.Type == models.TaskTypeStartServer || t.Type == models.TaskTypeOneClickStartGame) && t.EndAt != nil {
			lastStartEnd = t.EndAt
		} else if t.Type == models.TaskTypeArchive && t.EndAt != nil && lastStartEnd != nil {
			totalRuntime += t.EndAt.Sub(*lastStartEnd)
			lastStartEnd = nil
		}
	}
	stats.TotalRuntimeSec = int64(totalRuntime.Seconds())

	// 按任务类别统计最近完成时间
	type typeLastCompleted struct {
		Type  models.TaskType
		EndAt *time.Time
	}
	var typeResults []typeLastCompleted
	DB.Model(&models.Task{}).
		Select("type, MAX(end_at) as end_at").
		Where("status = ?", successStatus).
		Group("type").
		Find(&typeResults)
	for _, r := range typeResults {
		stats.LastCompletedByType[r.Type] = r.EndAt
	}

	return stats, nil
}

// GetTask 根据任务ID获取任务详情，并从 task_output_lines 拼接完整的输出文本。
func GetTask(taskId uint) (*models.Task, error) {
	task, err := GetTaskRecord(taskId)
	if err != nil {
		return nil, err
	}

	output, err := GetTaskOutputString(taskId)
	if err != nil {
		return nil, err
	}
	task.Output = output

	return task, nil
}

// GetTaskRecord 根据任务ID获取任务记录，不加载旧的 output 列。Preload 用户信息。
func GetTaskRecord(taskId uint) (*models.Task, error) {
	var task models.Task
	result := DB.Preload("User").Omit("output").First(&task, taskId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

// AppendTaskOutputLines 将一批任务输出行写入数据库。
func AppendTaskOutputLines(lines []models.TaskOutputLine) error {
	if len(lines) == 0 {
		return nil
	}
	return DB.Create(&lines).Error
}

// GetTaskOutputLinesRange 按序号升序返回任务输出行。
// 返回区间为 (afterSeq, maxSeq]；maxSeq 为 nil 表示不设上限。
func GetTaskOutputLinesRange(taskId uint, afterSeq uint64, maxSeq *uint64, limit int) ([]models.TaskOutputLine, error) {
	var lines []models.TaskOutputLine

	query := DB.Where("task_id = ? AND seq > ?", taskId, afterSeq)
	if maxSeq != nil {
		query = query.Where("seq <= ?", *maxSeq)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	result := query.Order("seq ASC").Find(&lines)
	return lines, result.Error
}

// GetLastTaskOutputSeq 返回任务已落库的最大输出序号。没有输出时返回 0。
func GetLastTaskOutputSeq(taskId uint) (uint64, error) {
	var result struct {
		MaxSeq uint64
	}

	query := DB.Model(&models.TaskOutputLine{}).
		Where("task_id = ?", taskId).
		Select("COALESCE(MAX(seq), 0) AS max_seq").
		Scan(&result)

	return result.MaxSeq, query.Error
}

// GetTaskOutputString 按行拼接任务的完整输出文本，行为与旧版 Task.Output 一致：
// 每一行（包括最后一行）后都带一个换行符。
func GetTaskOutputString(taskId uint) (string, error) {
	rows, err := DB.Model(&models.TaskOutputLine{}).
		Select("output").
		Where("task_id = ?", taskId).
		Order("seq ASC").
		Rows()
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var builder strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", err
		}
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String(), rows.Err()
}
