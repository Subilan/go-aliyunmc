package dao

import (
	"strings"
	"time"

	"github.com/Subilan/go-aliyunmc/internal/common/store"
	"github.com/Subilan/go-aliyunmc/internal/common/store/models"
	"github.com/Subilan/go-aliyunmc/internal/modules/lab/models"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CreateTask 初始化一个新的任务并立即写入到数据库。by 是创建任务的用户ID。
func CreateTask(taskType labmodels.TaskType, by *uint) (*labmodels.Task, error) {
	task := &labmodels.Task{
		Type:   taskType,
		Status: labmodels.TaskStatusCreated,
		Output: "",
		Error:  "",
		Step:   0,
		By:     by,
	}
	result := store.DB.Create(task)
	return task, result.Error
}

// UpdateTask 将任务的可变字段写入数据库。输出内容已拆分到 task_output_lines，
// 因此这里显式排除 output 列，避免每次落库都全量写入旧的大字段。
func UpdateTask(task *labmodels.Task) error {
	result := store.DB.Session(&gorm.Session{
		Logger: store.DB.Logger.LogMode(logger.Silent),
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
func ListTasks(status *labmodels.TaskStatus, sort, order string, limit, offset int) ([]labmodels.Task, error) {
	var tasks []labmodels.Task
	query := store.DB.Model(&labmodels.Task{})

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
func CountTasks(status *labmodels.TaskStatus) (int64, error) {
	var count int64
	query := store.DB.Model(&labmodels.Task{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	result := query.Count(&count)
	return count, result.Error
}

// TaskStats 包含任务概览统计数据。
type TaskStats struct {
	Total           int64        `json:"total"`
	SuccessCount    int64        `json:"successCount"`
	LastCompletedAt *time.Time   `json:"lastCompletedAt"`
	LastStartedAt   *time.Time   `json:"lastStartedAt"`
	LastStartedBy   *uint        `json:"lastStartedBy"`
	LastStartedUser *models.User `json:"lastStartedUser"`
	TotalRuntimeSec int64        `json:"totalRuntimeSec"`
}

// GetTaskStats 返回任务概览统计数据。
func GetTaskStats() (TaskStats, error) {
	var stats TaskStats

	if err := store.DB.Model(&labmodels.Task{}).Count(&stats.Total).Error; err != nil {
		return stats, err
	}

	successStatus := labmodels.TaskStatusSuccess
	if err := store.DB.Model(&labmodels.Task{}).Where("status = ?", successStatus).Count(&stats.SuccessCount).Error; err != nil {
		return stats, err
	}

	var lastCompleted labmodels.Task
	if err := store.DB.Where("status = ?", successStatus).Order("end_at DESC").First(&lastCompleted).Error; err == nil {
		stats.LastCompletedAt = lastCompleted.EndAt
	}

	// 最近一次成功启动游戏的任务，启动时间与发起人取自同一条记录。
	var lastStarted labmodels.Task
	if err := store.DB.Preload("User").
		Where("status = ? AND type IN ? AND end_at IS NOT NULL", successStatus, []labmodels.TaskType{
			labmodels.TaskTypeStartServer,
			labmodels.TaskTypeOneClickStartGame,
		}).
		Order("end_at DESC").First(&lastStarted).Error; err == nil {
		stats.LastStartedAt = lastStarted.EndAt
		stats.LastStartedBy = lastStarted.By
		stats.LastStartedUser = lastStarted.User
	}

	// 计算总运行时长：按 启动服务器—archive 配对。
	// oneclick_start_game 成功意味着服务器已经启动，因此也作为启动标记参与配对。
	var pairedTasks []labmodels.Task
	store.DB.Where("status = ? AND type IN ?", successStatus, []labmodels.TaskType{
		labmodels.TaskTypeStartServer,
		labmodels.TaskTypeOneClickStartGame,
		labmodels.TaskTypeArchive,
	}).Order("end_at ASC").Find(&pairedTasks)

	var totalRuntime time.Duration
	var lastStartEnd *time.Time
	for _, t := range pairedTasks {
		if (t.Type == labmodels.TaskTypeStartServer || t.Type == labmodels.TaskTypeOneClickStartGame) && t.EndAt != nil {
			lastStartEnd = t.EndAt
		} else if t.Type == labmodels.TaskTypeArchive && t.EndAt != nil && lastStartEnd != nil {
			totalRuntime += t.EndAt.Sub(*lastStartEnd)
			lastStartEnd = nil
		}
	}
	stats.TotalRuntimeSec = int64(totalRuntime.Seconds())

	return stats, nil
}

// GetTask 根据任务ID获取任务详情，并从 task_output_lines 拼接完整的输出文本。
func GetTask(taskId uint) (*labmodels.Task, error) {
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
func GetTaskRecord(taskId uint) (*labmodels.Task, error) {
	var task labmodels.Task
	result := store.DB.Preload("User").Omit("output").First(&task, taskId)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

// AppendTaskOutputLines 将一批任务输出行写入数据库。
func AppendTaskOutputLines(lines []labmodels.TaskOutputLine) error {
	if len(lines) == 0 {
		return nil
	}
	return store.DB.Create(&lines).Error
}

// GetTaskOutputLinesRange 按序号升序返回任务输出行。
// 返回区间为 (afterSeq, maxSeq]；maxSeq 为 nil 表示不设上限。
func GetTaskOutputLinesRange(taskId uint, afterSeq uint64, maxSeq *uint64, limit int) ([]labmodels.TaskOutputLine, error) {
	var lines []labmodels.TaskOutputLine

	query := store.DB.Where("task_id = ? AND seq > ?", taskId, afterSeq)
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

	query := store.DB.Model(&labmodels.TaskOutputLine{}).
		Where("task_id = ?", taskId).
		Select("COALESCE(MAX(seq), 0) AS max_seq").
		Scan(&result)

	return result.MaxSeq, query.Error
}

// GetTaskOutputString 按行拼接任务的完整输出文本，行为与旧版 Task.Output 一致：
// 每一行（包括最后一行）后都带一个换行符。
func GetTaskOutputString(taskId uint) (string, error) {
	rows, err := store.DB.Model(&labmodels.TaskOutputLine{}).
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
