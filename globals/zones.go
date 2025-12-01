package globals

import (
	"github.com/Subilan/gomc-server/config"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

type ZoneCacheItem struct {
	ZoneId                 *string   `json:"zoneId"`
	AvailableInstanceTypes []*string `json:"availableInstanceTypes"`
	LocalName              *string   `json:"localName"`
}

var ZoneCache []ZoneCacheItem

func GetZoneItemByZoneId(zoneId string) *ZoneCacheItem {
	for _, zone := range ZoneCache {
		if *zone.ZoneId == zoneId {
			return &zone
		}
	}

	return nil
}

func RetrieveZones(client *ecs20140526.Client) ([]ZoneCacheItem, error) {
	describeZonesRequest := &ecs20140526.DescribeZonesRequest{
		RegionId: tea.String(config.Cfg.Aliyun.RegionId),
	}

	describeZonesResponse, err := client.DescribeZones(describeZonesRequest)

	if err != nil {
		return nil, err
	}

	var result = make([]ZoneCacheItem, 0)

	for _, zone := range describeZonesResponse.Body.Zones.Zone {
		result = append(result, ZoneCacheItem{
			ZoneId:                 zone.ZoneId,
			AvailableInstanceTypes: zone.AvailableInstanceTypes.InstanceTypes,
			LocalName:              zone.LocalName,
		})
	}

	return result, nil
}
