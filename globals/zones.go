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

// GetZoneItemByZoneId 尝试从当前的可用区信息缓存中获取 zoneId 对应的 ZoneCacheItem，如果不存在返回 nil
func GetZoneItemByZoneId(zoneId string) *ZoneCacheItem {
	for _, zone := range ZoneCache {
		if *zone.ZoneId == zoneId {
			return &zone
		}
	}

	return nil
}

// IsInstanceTypeAvailableInZone 判断指定的实例类型是否存在于可用区信息缓存中 zoneId 指定的可用区的所有可用实例类型列表中
//
// 参见：ZoneCacheItem.AvailableInstanceTypes
func IsInstanceTypeAvailableInZone(instanceType string, zoneId string) bool {
	zone := GetZoneItemByZoneId(zoneId)

	if zone == nil {
		return false
	}

	for _, availableInstanceType := range zone.AvailableInstanceTypes {
		if *availableInstanceType == instanceType {
			return true
		}
	}

	return false
}

// RetrieveZones 从阿里云获取 config.toml 中设定的地域下的所有可用区的基本信息，以 ZoneCacheItem 数组的形式返回
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
