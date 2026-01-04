package store

import (
	"log"
	"time"

	"github.com/Subilan/go-aliyunmc/globals"
)

type Instance struct {
	InstanceId   string     `json:"instanceId"`
	InstanceType string     `json:"instanceType"`
	RegionId     string     `json:"regionId"`
	ZoneId       string     `json:"zoneId"`
	DeletedAt    *time.Time `json:"deletedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	Deployed     bool       `json:"deployed"`
	Ip           *string    `json:"ip"`
}

type InstanceStatus struct {
	InstanceId     string    `json:"instanceId"`
	InstanceStatus string    `json:"instanceStatus"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func GetInstanceStatus(instanceId string) *InstanceStatus {
	var result InstanceStatus

	err := globals.Pool.QueryRow("SELECT instance_id, status, updated_at FROM instance_statuses WHERE instance_id = ?", instanceId).Scan(&result.InstanceId, &result.InstanceStatus, &result.UpdatedAt)

	if err != nil {
		return nil
	}

	return &result
}

func getInstance(cond string) *Instance {
	var result Instance

	err := globals.Pool.QueryRow("SELECT instance_id, instance_type, region_id, zone_id, deleted_at, created_at, ip, deployed FROM instances "+cond).Scan(
		&result.InstanceId,
		&result.InstanceType,
		&result.RegionId,
		&result.ZoneId,
		&result.DeletedAt,
		&result.CreatedAt,
		&result.Ip,
		&result.Deployed,
	)

	if err != nil {
		return nil
	}

	return &result
}

// GetActiveInstance 从数据库获取当前的活动实例
// 如果找不到实例，或者活动实例没有分配IP地址，返回 nil
func GetActiveInstance() *Instance {
	result := getInstance("WHERE deleted_at IS NULL")

	if result == nil {
		return nil
	}

	if result.Ip == nil {
		log.Println("ip not allocated on active instance")
		return nil
	}

	return result
}

func GetLatestInstance() *Instance {
	result := getInstance("ORDER BY created_at DESC LIMIT 1")

	if result == nil {
		return nil
	}

	return result
}
