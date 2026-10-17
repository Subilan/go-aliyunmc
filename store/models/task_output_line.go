package models

import "time"

// TaskOutputLine 是任务输出的一行。任务输出按行拆分到独立表中，
// 避免每次落库都全量更新 tasks 表里的一个大文本字段。
type TaskOutputLine struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// TaskID 是输出所属的任务 ID。
	TaskID uint `gorm:"not null;uniqueIndex:idx_task_output_line_task_seq,priority:1" json:"taskId"`
	// Seq 是任务内单调递增的输出序号，从 1 开始，同时作为 SSE 事件 ID。
	Seq uint64 `gorm:"not null;uniqueIndex:idx_task_output_line_task_seq,priority:2" json:"seq"`
	// Step 是该行输出产生时任务所处的步骤。
	Step uint `gorm:"not null;default:0" json:"step"`
	// Output 是输出内容，不包含行尾换行符。
	Output string `gorm:"not null" json:"output"`
	// CreatedAt 是写入时间。
	CreatedAt time.Time `json:"createdAt"`
}
