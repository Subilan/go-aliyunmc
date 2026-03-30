package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskType string

const (
	// TaskTypeCreateAndDeploy 一键创建并部署服务器，部署完成后启动服务器，服务器启动完成后正常返回
	TaskTypeCreateAndDeploy TaskType = "create_and_deploy"

	// TaskTypeStartServer 启动服务器，当服务器启动完成后正常返回
	TaskTypeStartServer TaskType = "start_server"

	// TaskTypeStopServer 停止服务器，当服务器关闭完成后正常返回
	TaskTypeStopServer TaskType = "stop_server"
)

type TaskStatus string

const (
	TaskStatusCreated TaskStatus = "created"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
)

type Task struct {
	gorm.Model
	Type    TaskType   `gorm:"not null" json:"type"`
	StartAt *time.Time `gorm:"default:null" json:"startAt"`
	EndAt   *time.Time `gorm:"default:null" json:"endAt"`
	Status  TaskStatus `gorm:"not null" json:"status"`
	Step    uint       `gorm:"not null,default:0" json:"step"`
	Output  string     `gorm:"not null" json:"output"`
	Error   string     `gorm:"default:null" json:"error"`
	By      *uint      `gorm:"default:null" json:"by"`
	User    *User      `gorm:"foreignKey:By" json:"user"`
}
