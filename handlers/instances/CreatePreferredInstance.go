package instances

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/events"
	"github.com/Subilan/go-aliyunmc/events/stream"
	"github.com/Subilan/go-aliyunmc/helpers"
	"github.com/Subilan/go-aliyunmc/helpers/db"
	"github.com/Subilan/go-aliyunmc/helpers/store"
	"github.com/Subilan/go-aliyunmc/monitors"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/gin-gonic/gin"
)

// CreateInstanceQuery 是创建实例接口的请求体
type CreateInstanceQuery struct {
	// AutoVSwitch 表示是否在默认交换机不存在时，在指定可用区内自动创建默认交换机，并立即使用该交换机
	AutoVSwitch bool `json:"autoVSwitch" form:"autoVSwitch"`
}

type CreateInstanceResponseBody struct {
	InstanceId string  `json:"instanceId"`
	TradePrice float32 `json:"tradePrice"`
}

const createInstanceTimeout = 15 * time.Second

var createInstanceMutex sync.Mutex

func createPreferredInstance() helpers.QueryHandlerFunc[CreateInstanceQuery] {
	return func(query CreateInstanceQuery, c *gin.Context) (any, error) {
		// 长耗时任务，避免重复执行
		ok := createInstanceMutex.TryLock()
		if !ok {
			return nil, &helpers.HttpError{Code: http.StatusForbidden, Details: "instance is being created"}
		}
		defer createInstanceMutex.Unlock()

		if !monitors.SnapshotPreferredInstanceChargePresent() {
			return nil, &helpers.HttpError{Code: http.StatusServiceUnavailable, Details: "preferred instance charge not present"}
		}

		inst := monitors.SnapshotPreferredInstanceCharge()
		zoneId := inst.ZoneId
		instanceType := inst.InstanceType

		var err error

		describeVSwitchesRequest := &vpc20160428.DescribeVSwitchesRequest{
			ZoneId:    tea.String(zoneId),
			IsDefault: tea.Bool(true),
		}

		describeVSwitchesResponse, err := clients.VpcClient.DescribeVSwitches(describeVSwitchesRequest)

		if err != nil {
			return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: "cannot describe vswitches in zone " + zoneId}
		}

		var vswitchId string

		if len(describeVSwitchesResponse.Body.VSwitches.VSwitch) == 0 {
			if !query.AutoVSwitch {
				return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "vswitch not found"}
			}
			createDefaultVSwitchRequest := &vpc20160428.CreateDefaultVSwitchRequest{
				ZoneId:   tea.String(zoneId),
				RegionId: tea.String(config.Cfg.Aliyun.RegionId),
			}

			createDefaultVSwitchResponse, err := clients.VpcClient.CreateDefaultVSwitch(createDefaultVSwitchRequest)

			if err != nil {
				return nil, &helpers.HttpError{Code: http.StatusInternalServerError, Details: "cannot create default vswitch in zone " + zoneId}
			}

			// 此时交换机可能仍然在准备中
			vswitchId = *createDefaultVSwitchResponse.Body.VSwitchId
		} else {
			vswitchId = *describeVSwitchesResponse.Body.VSwitches.VSwitch[0].VSwitchId
		}

		ctx, cancel := context.WithTimeout(c, createInstanceTimeout)
		defer cancel()

		var cnt int

		err = db.Pool.QueryRowContext(ctx, "SELECT count(*) FROM instances WHERE deleted_at IS NULL").Scan(&cnt)

		if err != nil {
			return nil, err
		}

		if cnt > 0 {
			return nil, &helpers.HttpError{Code: http.StatusConflict, Details: "an instance already exists"}
		}

		ecsConfig := config.Cfg.GetAliyunEcsConfig()

		createInstanceRequest := &ecs20140526.CreateInstanceRequest{
			RegionId:     tea.String(config.Cfg.Aliyun.RegionId),
			ZoneId:       &zoneId,
			InstanceType: tea.String(instanceType),
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
			InternetChargeType:       tea.String("PayByTraffic"), // This line costs CNY 400.
			InternetMaxBandwidthOut:  tea.Int32(int32(ecsConfig.InternetMaxBandwidthOut)),
			HostName:                 tea.String(ecsConfig.HostName),
			Password:                 tea.String(ecsConfig.RootPassword),
			InstanceChargeType:       tea.String("PostPaid"),
			SpotStrategy:             tea.String("SpotAsPriceGo"),
			SpotDuration:             tea.Int32(1),
			SpotInterruptionBehavior: tea.String(ecsConfig.SpotInterruptionBehavior),
			SecurityGroupId:          tea.String(ecsConfig.SecurityGroupId),
			VSwitchId:                tea.String(vswitchId),
			ImageId:                  tea.String(ecsConfig.ImageId),
		}

		createInstanceResponse, err := clients.EcsClient.CreateInstance(createInstanceRequest)

		if err != nil {
			return nil, err
		}

		_, err = db.Pool.ExecContext(ctx, `
INSERT INTO instances (instance_id, instance_type, region_id, zone_id, vswitch_id) VALUES (?, ?, ?, ?, ?)
`, *createInstanceResponse.Body.InstanceId, instanceType, config.Cfg.Aliyun.RegionId, zoneId, vswitchId)

		if err != nil {
			return nil, err
		}

		// 将实例创建广播给所有用户
		event := events.Instance(events.InstanceEventCreated, store.Instance{
			InstanceId:   *createInstanceResponse.Body.InstanceId,
			InstanceType: instanceType,
			RegionId:     config.Cfg.Aliyun.RegionId,
			ZoneId:       zoneId,
			DeletedAt:    nil,
			CreatedAt:    time.Now(),
			Ip:           nil,
			VSwitchId:    vswitchId,
		}, true)
		err = stream.BroadcastAndSave(event)

		if err != nil {
			log.Println("cannot broadcast and save event:", err)
		}

		go monitors.StartActiveInstanceWhenReady()

		return gin.H{}, nil
	}
}

func HandleCreatePreferredInstance() gin.HandlerFunc {
	return helpers.QueryHandler(createPreferredInstance())
}
