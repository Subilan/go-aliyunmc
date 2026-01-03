package instances

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/helpers/stream"
	"github.com/Subilan/go-aliyunmc/monitors"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// CreateInstanceBody 是创建实例接口的请求体
type CreateInstanceBody struct {
	// ZoneId 是当前希望创建的实例所在的可用区 ID，例如 cn-shenzhen-c
	ZoneId string `json:"zoneId" binding:"required"`

	// VSwitchId 是当前希望创建实例使用的交换机 ID，其必须位于 ZoneId 表示的可用区内
	VSwitchId string `json:"vSwitchId" binding:"required"`

	// InstanceType 是当前希望创建实例的类型
	InstanceType string `json:"instanceType" binding:"required"`

	// DryRun 表示当前是否为预检请求
	DryRun bool `json:"dryRun"`
}

type CreateInstanceResponseBody struct {
	InstanceId string  `json:"instanceId"`
	TradePrice float32 `json:"tradePrice"`
}

const CreateInstanceTimeout = 15 * time.Second

var createInstanceMutex sync.Mutex

func HandleCreateInstance() gin.HandlerFunc {
	return helpers.BodyHandler(func(body CreateInstanceBody, c *gin.Context) (any, error) {
		// 长耗时任务，避免重复执行
		createInstanceMutex.Lock()
		defer createInstanceMutex.Unlock()

		var err error

		zone := globals.GetZoneItemByZoneId(body.ZoneId)

		if zone == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "zone not found"}
		}

		if !globals.IsInstanceTypeAvailableInZone(body.InstanceType, body.ZoneId) {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: fmt.Sprintf("instance type %s not available in zone %s or zone does not exist", body.InstanceType, body.ZoneId)}
		}

		if !globals.IsVSwitchInZone(body.VSwitchId, body.ZoneId) {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: fmt.Sprintf("vSwitch %s not found in region %s", body.VSwitchId, body.ZoneId)}
		}

		ctx, cancel := context.WithTimeout(c, CreateInstanceTimeout)
		defer cancel()

		var cnt int

		err = globals.Pool.QueryRowContext(ctx, "SELECT count(*) FROM instances WHERE deleted_at IS NULL").Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt > 0 {
			return nil, &helpers.HttpError{Code: http.StatusConflict, Details: "an instance already exists"}
		}

		ecsConfig := config.Cfg.GetAliyunEcsConfig()

		createInstanceRequest := &ecs20140526.CreateInstanceRequest{
			RegionId:     tea.String(config.Cfg.Aliyun.RegionId),
			ZoneId:       zone.ZoneId,
			InstanceType: tea.String(body.InstanceType),
			SystemDisk: &ecs20140526.CreateInstanceRequestSystemDisk{
				Category: tea.String(ecsConfig.SystemDisk.Category),
				Size:     tea.Int32(int32(ecsConfig.SystemDisk.Size)),
			},
			DataDisk: []*ecs20140526.CreateInstanceRequestDataDisk{
				{
					Category: tea.String(ecsConfig.DataDisk.Category),
					Size:     tea.Int32(int32(ecsConfig.DataDisk.Size)),
					DiskName: tea.String("data"),
				},
			},
			InternetChargeType:       tea.String("PayByBandwidth"),
			InternetMaxBandwidthOut:  tea.Int32(int32(ecsConfig.InternetMaxBandwidthOut)),
			HostName:                 tea.String(ecsConfig.HostName),
			Password:                 tea.String(ecsConfig.RootPassword),
			InstanceChargeType:       tea.String("PostPaid"),
			SpotStrategy:             tea.String("SpotAsPriceGo"),
			SpotDuration:             tea.Int32(1),
			SpotInterruptionBehavior: tea.String(ecsConfig.SpotInterruptionBehavior),
			DryRun:                   tea.Bool(body.DryRun),
			SecurityGroupId:          tea.String(ecsConfig.SecurityGroupId),
			VSwitchId:                tea.String(body.VSwitchId),
			ImageId:                  tea.String(ecsConfig.ImageId),
		}

		createInstanceResponse, err := globals.EcsClient.CreateInstance(createInstanceRequest)

		if err != nil {
			return nil, err
		}

		tx, err := globals.Pool.BeginTx(ctx, nil)

		if err != nil {
			return nil, err
		}

		_, err = tx.ExecContext(ctx, `
INSERT INTO instances (instance_id, instance_type, region_id, zone_id) VALUES (?, ?, ?, ?)
`, *createInstanceResponse.Body.InstanceId, body.InstanceType, config.Cfg.Aliyun.RegionId, body.ZoneId)

		if err != nil {
			tx.Rollback()
			return nil, err
		}

		_, err = tx.ExecContext(ctx, `
INSERT INTO instance_statuses (instance_id, status) VALUES (?, ?)
`, *createInstanceResponse.Body.InstanceId, "__created__")

		if err != nil {
			tx.Rollback()
			return nil, err
		}

		err = tx.Commit()

		if err != nil {
			return nil, err
		}

		// 将实例创建广播给所有用户
		event, err := store.BuildInstanceEvent(store.InstanceEventCreated, store.Instance{
			InstanceId:   *createInstanceResponse.Body.InstanceId,
			InstanceType: body.InstanceType,
			RegionId:     config.Cfg.Aliyun.RegionId,
			ZoneId:       body.ZoneId,
			DeletedAt:    nil,
			CreatedAt:    time.Now(),
			Ip:           nil,
		})

		if err != nil {
			log.Println("cannot build event:", err)
		}

		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast and save event:", err)
		}

		go monitors.StartInstanceWhenReady()

		return gin.H{}, nil
	})
}
