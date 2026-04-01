package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskType string

const (
	// TaskTypeCreateAndDeploy 是一键创建并部署服务器任务，它在部署完成后启动服务器，服务器启动完成后正常返回。
	TaskTypeCreateAndDeploy TaskType = "create_and_deploy"

	// TaskTypeStartServer 是启动服务器任务。当服务器启动完成后正常返回。
	TaskTypeStartServer TaskType = "start_server"

	// TaskTypeStopServer 是停止服务器任务。当服务器关闭完成后正常返回。
	TaskTypeStopServer TaskType = "stop_server"
)

type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
)

// Task 模型表示一个异步执行的任务。它包含了任务的类型、状态、输出、错误信息以及与创建任务的用户的关联等字段。
type Task struct {
	gorm.Model
	// Type 字段表示任务的类型，它是一个字符串，用于唯一标识一种任务。Type 与任务函数 F 一一对应，可以理解为任务函数的标识符。
	Type TaskType `gorm:"not null" json:"type"`
	// StartAt 字段记录了任务的开始时间。当任务开始执行时，StartAt 将被设置为任务开始的时间戳。对于尚未开始的任务，StartAt 将保持为空。
	StartAt *time.Time `gorm:"default:null" json:"startAt"`
	// EndAt 字段记录了任务的结束时间。当任务完成（无论成功还是失败）时，EndAt 将被设置为任务结束的时间戳。对于正在运行或尚未开始的任务，EndAt 将保持为空。
	EndAt *time.Time `gorm:"default:null" json:"endAt"`
	// Status 字段表示任务的当前状态。它是一个字符串，通常具有以下几个可能的值：
	//  - created：表示任务已创建但尚未开始执行。
	//  - running：表示任务正在执行中。
	//  - success：表示任务已成功完成。
	//  - failed：表示任务执行失败。
	Status TaskStatus `gorm:"not null" json:"status"`
	// Step 字段表示任务的当前步骤，通常用于多步骤任务，以便前端能够根据步骤信息展示不同的界面或提示用户当前任务的进度。
	// 对于单步骤任务，Step 保持为 0。
	Step uint `gorm:"not null,default:0" json:"step"`
	// Output 字段记录了任务执行过程中的输出信息。它可以包含多行文本，描述任务的执行进度、结果或任何相关信息。
	Output string `gorm:"not null" json:"output"`
	// Error 字段记录了任务执行失败时的错误信息。它可以是一个简短的错误描述，也可能是特殊的字符串：
	//  - __TIMEOUT__：表示任务因超时而失败。
	//  - __INTERRUPTED__：表示任务被中断而失败。
	//  - 空字符串：表示任务没有错误（成功或正在运行中）。
	Error string `gorm:"default:null" json:"error"`
	// By 字段记录了创建任务的用户ID。如果为空，表示该任务由系统创建。
	By *uint `gorm:"default:null" json:"by"`
	// User 字段是一个关联字段，表示创建任务的用户信息。它通过外键 By 关联到 User 模型。当 By 为空时，User 也将为 nil。
	User *User `gorm:"foreignKey:By" json:"user"`
}
