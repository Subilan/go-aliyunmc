package helpers

import (
	"github.com/Subilan/gomc-server/config"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

type ZoneItem struct {
	ZoneId                 *string   `json:"zoneId"`
	AvailableInstanceTypes []*string `json:"availableInstanceTypes"`
	LocalName              *string   `json:"localName"`
}

func RetrieveZones(client *ecs20140526.Client) ([]ZoneItem, error) {
	describeZonesRequest := &ecs20140526.DescribeZonesRequest{
		RegionId: tea.String(config.Cfg.Aliyun.Ecs.RegionId),
	}

	describeZonesResponse, err := client.DescribeZones(describeZonesRequest)

	if err != nil {
		return nil, err
	}

	var result = make([]ZoneItem, 0)

	for _, zone := range describeZonesResponse.Body.Zones.Zone {
		result = append(result, ZoneItem{
			ZoneId:                 zone.ZoneId,
			AvailableInstanceTypes: zone.AvailableInstanceTypes.InstanceTypes,
			LocalName:              zone.LocalName,
		})
	}

	return result, nil
}
