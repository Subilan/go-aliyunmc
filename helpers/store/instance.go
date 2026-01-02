package store

import (
	"time"

	"github.com/Subilan/gomc-server/globals"
)

type Instance struct {
	InstanceId   string     `json:"instanceId"`
	InstanceType string     `json:"instanceType"`
	RegionId     string     `json:"regionId"`
	ZoneId       string     `json:"zoneId"`
	DeletedAt    *time.Time `json:"deletedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	Ip           string     `json:"ip"`
}

func GetActiveInstance() *Instance {
	var result Instance

	err := globals.Pool.QueryRow("SELECT instance_id, instance_type, region_id, zone_id, deleted_at, created_at, ip FROM instances WHERE deleted_at IS NULL").Scan(
		&result.InstanceId,
		&result.InstanceType,
		&result.RegionId,
		&result.ZoneId,
		&result.DeletedAt,
		&result.CreatedAt,
		&result.Ip,
	)

	if err != nil {
		return nil
	}

	return &result
}

// GetRunningInstanceBrief 尝试获取一个处于运行状态的实例并返回其instance_id和ip信息
func GetRunningInstanceBrief() (string, string, error) {
	var instanceId, ip string
	err := globals.Pool.QueryRow("SELECT i.instance_id, i.ip FROM instances i JOIN instance_statuses s ON i.instance_id = s.instance_id WHERE i.ip IS NOT NULL AND i.deleted_at IS NULL AND s.status = 'Running'").Scan(&instanceId, &ip)

	if err != nil {
		return "", "", err
	}

	return instanceId, ip, nil
}
