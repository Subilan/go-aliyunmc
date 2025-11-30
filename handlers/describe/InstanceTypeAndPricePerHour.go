package describe

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/Subilan/gomc-server/clients"
	"github.com/Subilan/gomc-server/config"
	"github.com/Subilan/gomc-server/globals"
	"github.com/Subilan/gomc-server/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

// InstanceTypeAndPricePerHourBody is the request structure of this handler.
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

// InstanceTypeAndPricePerHourResponseItem is the response structure of this handler.
// For the crucial part, see InstanceTypeAndTradePrice.
type InstanceTypeAndPricePerHourResponseItem struct {
	ZoneId             string                      `json:"zoneId"`
	TypesAndTradePrice []InstanceTypeAndTradePrice `json:"typesAndTradePrice"`
}

// InstanceTypeAndTradePrice describes an instance with its fundamental configurations (e.g. MemorySize and CpuCoreCount) and its TradePrice under SpotAsPriceGo spot strategy.
// There might be a string Comment attached to indicate the reason of its possible unusual construction,
// for example, there might be no informative field due to the fact that they cannot be retrieved as a whole under the given ECS settings defined in config.toml.
type InstanceTypeAndTradePrice struct {
	// InstanceType, or instance type id, identifies an ECS instance type.
	InstanceType string `json:"instanceType"`

	// CpuCoreCount is the count of vCPU
	CpuCoreCount int32 `json:"cpuCoreCount,omitempty"`

	// MemorySize is the memory size in GiB
	MemorySize float32 `json:"memorySize,omitempty"`

	// TradePrice is the price in CNY of this type of ECS instance, calculated under SpotAsPriceGo spot strategy and current ECS settings defined in config.toml.
	TradePrice float32 `json:"tradePrice,omitempty"`

	// Comment describes the detailed reason when the fields cannot be retrieved as a whole.
	Comment string `json:"comment,omitempty"`
}

func InstanceTypeAndPricePerHour() helpers.QueryHandlerFunc[InstanceTypeAndPricePerHourBody] {
	return func(body InstanceTypeAndPricePerHourBody, c *gin.Context) (any, error) {
		client, err := clients.ShouldCreateEcsClient()
		ecsConfig := config.Cfg.GetAliyunEcsConfig()

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

		var targetZoneItems []helpers.ZoneItem

		if body.ZoneId != "" {
			onlyZone := globals.GetZoneItemByZoneId(body.ZoneId)

			if onlyZone == nil {
				return nil, &helpers.HttpError{Code: http.StatusNotFound, Details: "zone id not found"}
			}

			// TODO: could be multiple
			targetZoneItems = []helpers.ZoneItem{
				*onlyZone,
			}
		} else {
			targetZoneItems = globals.Zones
		}

		for _, zoneItem := range targetZoneItems {
			var typesAndTradePrice = make([]InstanceTypeAndTradePrice, 0)
			zoneAvailableFilteredInstanceTypes := helpers.IntersectHashGeneric(tea.StringSliceValue(zoneItem.AvailableInstanceTypes), instanceTypeIds)

			for _, instanceType := range zoneAvailableFilteredInstanceTypes {
				describePriceRequest := &ecs20140526.DescribePriceRequest{
					RegionId:                tea.String(ecsConfig.RegionId),
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

				describePriceResponse, err := client.DescribePrice(describePriceRequest)

				currentTypeAndTradePrice := InstanceTypeAndTradePrice{
					InstanceType: instanceType,
				}

				if err != nil {
					fmt.Printf("cannot retrieve price for ecs type [%s] under region [%s] zone [%s]\n", instanceType, ecsConfig.RegionId, *zoneItem.ZoneId)
					currentTypeAndTradePrice.Comment = err.Error()
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
	}
}
