package ecsActions

import (
	"net/http"

	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
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

func CreateInstance() helpers.BodyHandlerFunc[CreateInstanceBody] {
	return func(body CreateInstanceBody, c *gin.Context) (any, error) {
		zone := globals.GetZoneItemByZoneId(body.ZoneId)

		if zone == nil {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "zone not found"}
		}

		found := false

		for _, availableType := range zone.AvailableInstanceTypes {
			if *availableType == body.InstanceType {
				found = true
				break
			}
		}

		if !found {
			return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "instance type not found"}
		}

		ecsConfig := config.Cfg.GetAliyunEcsConfig()

		createInstanceRequest := &ecs20140526.CreateInstanceRequest{
			RegionId:     tea.String(ecsConfig.RegionId),
			ZoneId:       zone.ZoneId,
			InstanceType: tea.String(body.InstanceType),
			SystemDisk: &ecs20140526.CreateInstanceRequestSystemDisk{
				Category: tea.String(ecsConfig.SystemDisk.Category),
				Size:     tea.Int32(int32(ecsConfig.SystemDisk.Size)),
			},
			DataDisk: []*ecs20140526.CreateInstanceRequestDataDisk{
				{
					Category: tea.String(ecsConfig.SystemDisk.Category),
					Size:     tea.Int32(int32(ecsConfig.SystemDisk.Size)),
					DiskName: tea.String("data"),
				},
			},
			InternetChargeType:       tea.String("PayByBandwidth"),
			InternetMaxBandwidthOut:  tea.Int32(int32(ecsConfig.InternetMaxBandwidthOut)),
			HostName:                 tea.String(ecsConfig.HostName),
			Password:                 tea.String(ecsConfig.Password),
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

		return helpers.Data(CreateInstanceResponseBody{
			InstanceId: tea.StringValue(createInstanceResponse.Body.InstanceId),
			TradePrice: tea.Float32Value(createInstanceResponse.Body.TradePrice),
		}), nil
	}
}
