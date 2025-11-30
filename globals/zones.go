package globals

import (
	"github.com/Subilan/gomc-server/helpers"
)

var Zones []helpers.ZoneItem

func GetZoneItemByZoneId(zoneId string) *helpers.ZoneItem {
	for _, zone := range Zones {
		if *zone.ZoneId == zoneId {
			return &zone
		}
	}

	return nil
}
