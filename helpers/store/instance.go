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
