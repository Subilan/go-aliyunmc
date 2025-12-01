package globals

import vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"

type VSwitchCacheItem struct {
	ZoneId    string `toml:"zoneId"`
	VSwitchId string `toml:"vSwitchId"`
}

var VSwitchCache []VSwitchCacheItem

// IsVSwitchInZone 判断当前的交换机信息缓存中，是否存在 zoneId 指定的可用区下标识符为 vSwitchId 的交换机
func IsVSwitchInZone(vSwitchId string, zoneId string) bool {
	for _, vSwitchItem := range VSwitchCache {
		if vSwitchItem.ZoneId == zoneId && vSwitchItem.VSwitchId == vSwitchId {
			return true
		}
	}

	return false
}

func RetrieveVSwitches(client *vpc20160428.Client) ([]VSwitchCacheItem, error) {
	describeVSwitchesRequest := &vpc20160428.DescribeVSwitchesRequest{}

	describeVSwitchesResponse, err := client.DescribeVSwitches(describeVSwitchesRequest)

	if err != nil {
		return nil, err
	}

	var result = make([]VSwitchCacheItem, 0)

	for _, vSwitch := range describeVSwitchesResponse.Body.VSwitches.VSwitch {
		result = append(result, VSwitchCacheItem{
			ZoneId:    *vSwitch.ZoneId,
			VSwitchId: *vSwitch.VSwitchId,
		})
	}

	return result, nil
}
