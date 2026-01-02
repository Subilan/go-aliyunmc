package instances

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// InstanceTypeAndPricePerHourBody 是查询实例类型及价格接口的请求体结构
type InstanceTypeAndPricePerHourBody struct {
	ZoneId              string `json:"zoneId" form:"zoneId"`
	MinimumMemorySize   int    `json:"minimumMemorySize,omitempty" form:"minimumMemorySize" binding:"required"`
	MaximumMemorySize   int    `json:"maximumMemorySize,omitempty" form:"maximumMemorySize" binding:"required"`
	MinimumCpuCoreCount int    `json:"minimumCpuCoreCount,omitempty" form:"minimumCpuCoreCount" binding:"required"`
	MaximumCpuCoreCount int    `json:"maximumCpuCoreCount,omitempty" form:"maximumCpuCoreCount" binding:"required"`
	CpuArchitecture     string `json:"cpuArchitecture,omitempty" form:"cpuArchitecture" binding:"required" validate:"oneof=X86 ARM"`
	helpers.Paginated
	helpers.Sorted
}

// InstanceTypeAndPricePerHourResponseItem 是查询实例类型及价格接口的返回结构
type InstanceTypeAndPricePerHourResponseItem struct {
	// ZoneId 是可用区的代码，例如cn-shenzhen-c
	ZoneId string `json:"zoneId"`

	// TypesAndTradePrice 是查询的结果数组
	TypesAndTradePrice []InstanceTypeAndTradePrice `json:"typesAndTradePrice"`
}

// InstanceTypeAndTradePrice 用来表示一个实例的类型、配置信息和成交价格
type InstanceTypeAndTradePrice struct {
	// InstanceType 是该实例的类型代码，例如 ecs.g6.xlarge
	InstanceType string `json:"instanceType"`

	// CpuCoreCount 是该实例的CPU核数
	CpuCoreCount int32 `json:"cpuCoreCount,omitempty"`

	// MemorySize 是该实例的内存大小，单位GiB
	MemorySize float32 `json:"memorySize,omitempty"`

	// TradePrice 是该实例在抢占式竞价策略下的每小时成交价格，单位CNY
	TradePrice float32 `json:"tradePrice,omitempty"`
}

func HandleGetInstanceTypesAndSpotPricePerHour() gin.HandlerFunc {
	return helpers.QueryHandler(func(body InstanceTypeAndPricePerHourBody, c *gin.Context) (any, error) {
		client, err := clients.ShouldCreateEcsClient()
		ecsConfig := config.Cfg.GetAliyunEcsConfig()
		regionId := config.Cfg.Aliyun.RegionId

		if err != nil {
			return nil, err
		}

		describeInstanceTypesRequest := &ecs20140526.DescribeInstanceTypesRequest{
			MaximumMemorySize:   tea.Float32(float32(body.MaximumMemorySize)),
			MinimumMemorySize:   tea.Float32(float32(body.MinimumMemorySize)),
			MaximumCpuCoreCount: tea.Int32(int32(body.MaximumCpuCoreCount)),
			MinimumCpuCoreCount: tea.Int32(int32(body.MinimumMemorySize)),
			CpuArchitecture:     tea.String(body.CpuArchitecture),
			NextToken:           body.NextToken,
		}

		if body.MaxResults != nil {
			describeInstanceTypesRequest.MaxResults = body.MaxResults
		}

		// 获取地域下所有满足需要的实例类型
		describeInstanceTypesResponse, err := client.DescribeInstanceTypes(describeInstanceTypesRequest)

		if err != nil {
			return nil, err
		}

		var instanceTypeIds = make([]string, 0)
		var instanceTypeInfoMap = make(map[string]*ecs20140526.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType)

		for _, inst := range describeInstanceTypesResponse.Body.InstanceTypes.InstanceType {
			instanceTypeIds = append(instanceTypeIds, *inst.InstanceTypeId)
			instanceTypeInfoMap[*inst.InstanceTypeId] = inst
		}

		var result = make([]InstanceTypeAndPricePerHourResponseItem, 0)

		var targetZoneItems []globals.ZoneCacheItem

		// 若请求体指定了可用区，则只在该可用区内搜索；否则在当前地域的所有可用区搜索
		if body.ZoneId != "" {
			onlyZone := globals.GetZoneItemByZoneId(body.ZoneId)

			if onlyZone == nil {
				return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "zone id not found"}
			}

			// TODO: could be multiple
			targetZoneItems = []globals.ZoneCacheItem{
				*onlyZone,
			}
		} else {
			targetZoneItems = globals.ZoneCache
		}

		// 对当前地域的每一个可用区
		for _, zoneItem := range targetZoneItems {
			var typesAndTradePrice = make([]InstanceTypeAndTradePrice, 0)

			// 找出当前可用区内的实例类型与按参数搜索结果中的实例类型的交集
			zoneAvailableFilteredInstanceTypes := helpers.IntersectHashGeneric(tea.StringSliceValue(zoneItem.AvailableInstanceTypes), instanceTypeIds)

			// 对当前可用区满足要求的每一个实例类型
			for _, instanceType := range zoneAvailableFilteredInstanceTypes {
				describePriceRequest := &ecs20140526.DescribePriceRequest{
					RegionId:                tea.String(regionId),
					ResourceType:            tea.String("instance"),
					InstanceType:            tea.String(instanceType),
					InternetChargeType:      tea.String("PayByTraffic"),
					InternetMaxBandwidthOut: tea.Int32(int32(ecsConfig.InternetMaxBandwidthOut)),
					SystemDisk: &ecs20140526.DescribePriceRequestSystemDisk{
						Category: tea.String(ecsConfig.SystemDisk.Category),
						Size:     tea.Int32(int32(ecsConfig.SystemDisk.Size)),
					},
					DataDisk: []*ecs20140526.DescribePriceRequestDataDisk{
						{
							Category: tea.String(ecsConfig.DataDisk.Category),
							Size:     tea.Int64(int64(ecsConfig.DataDisk.Size)),
						},
					},
					ZoneId:       zoneItem.ZoneId,
					SpotStrategy: tea.String("SpotAsPriceGo"),
					SpotDuration: tea.Int32(1),
				}

				// 获取价格
				// Note: 由于查价接口传入的信息相比查询实例类型接口传入的信息多出了对系统盘和数据盘的参数
				// 此处可能因为系统盘、数据盘的配置在指定实例上不受支持而报错
				describePriceResponse, err := client.DescribePrice(describePriceRequest)

				currentTypeAndTradePrice := InstanceTypeAndTradePrice{
					InstanceType: instanceType,
				}

				if err != nil {
					//fmt.Printf("warn: cannot retrieve price for ecs type [%s] under region [%s] zone [%s]\n", instanceType, regionId, *zoneItem.ZoneId)
					continue
				} else {
					currentTypeAndTradePrice.TradePrice = *describePriceResponse.Body.PriceInfo.Price.TradePrice
				}

				if currentTypeAndTradePrice.TradePrice > 0 {
					info := instanceTypeInfoMap[instanceType]

					currentTypeAndTradePrice.CpuCoreCount = *info.CpuCoreCount
					currentTypeAndTradePrice.MemorySize = *info.MemorySize
				}

				typesAndTradePrice = append(typesAndTradePrice, currentTypeAndTradePrice)
			}

			result = append(result, InstanceTypeAndPricePerHourResponseItem{
				ZoneId:             *zoneItem.ZoneId,
				TypesAndTradePrice: typesAndTradePrice,
			})
		}

		// 对结果按照指定字段进行排序
		if body.SortBy != nil {
			var sortBy = *body.SortBy
			validFields := []string{"cpuCoreCount", "memorySize", "tradePrice"}

			if slices.Contains(validFields, sortBy) {
				var sortOrder = "asc"

				if body.SortOrder != nil {
					sortOrder = strings.ToLower(*body.SortOrder)
				}

				for _, item := range result {
					typesAndTradePrice := item.TypesAndTradePrice

					sort.SliceStable(typesAndTradePrice, func(i, j int) bool {
						a := typesAndTradePrice[i]
						b := typesAndTradePrice[j]

						var less bool
						switch sortBy {
						case "cpuCoreCount":
							if a.CpuCoreCount == b.CpuCoreCount {
								less = false
							} else {
								less = a.CpuCoreCount < b.CpuCoreCount
							}
						case "memorySize":
							if a.MemorySize == b.MemorySize {
								less = false
							} else {
								less = a.MemorySize < b.MemorySize
							}
						case "tradePrice":
							if a.TradePrice == b.TradePrice {
								less = false
							} else {
								less = a.TradePrice < b.TradePrice
							}
						default:
							less = false
						}

						if sortOrder == "desc" {
							return !less
						}
						return less
					})
				}
			}
		}

		return helpers.Data(result), nil
	})
}
