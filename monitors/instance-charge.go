package monitors

import (
	"context"
	"errors"
	"log"
	"os"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/config"
	"github.com/Subilan/go-aliyunmc/consts"
	"github.com/Subilan/go-aliyunmc/globals"
	"github.com/Subilan/go-aliyunmc/helpers"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

// InstanceChargeItem 是查询实例类型及价格接口的返回结构
type InstanceChargeItem struct {
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

//type InstanceQueryOptions struct {
//	ZoneId              string `json:"zoneId" form:"zoneId"`
//	MinimumMemorySize   int    `json:"minimumMemorySize,omitempty" form:"minimumMemorySize" binding:"required"`
//	MaximumMemorySize   int    `json:"maximumMemorySize,omitempty" form:"maximumMemorySize" binding:"required"`
//	MinimumCpuCoreCount int    `json:"minimumCpuCoreCount,omitempty" form:"minimumCpuCoreCount" binding:"required"`
//	MaximumCpuCoreCount int    `json:"maximumCpuCoreCount,omitempty" form:"maximumCpuCoreCount" binding:"required"`
//	CpuArchitecture     string `json:"cpuArchitecture,omitempty" form:"cpuArchitecture" binding:"required" validate:"oneof=X86 ARM"`
//	SortBy              string `json:"sortBy,omitempty" form:"sortBy"`
//}

type IntRange struct {
	Min int
	Max int
}

// GetInstanceCharge 尝试获取指定可用区下，符合要求的所有实例类型，并获取该实例类型在抢占式实例中的每小时预估价格。
func GetInstanceCharge(
	ctx context.Context,
	zoneId string,
	memRange IntRange,
	cpuCoreCountRange IntRange,
	sortBy consts.InstanceChargeSortBy,
	sortOrder consts.SortOrder,
	maxResults int64,
) ([]InstanceChargeItem, error) {
	ecsConfig := config.Cfg.GetAliyunEcsConfig()
	regionId := config.Cfg.Aliyun.RegionId

	describeInstanceTypesRequest := &ecs20140526.DescribeInstanceTypesRequest{
		MaximumMemorySize:   tea.Float32(float32(memRange.Max)),
		MinimumMemorySize:   tea.Float32(float32(memRange.Min)),
		MaximumCpuCoreCount: tea.Int32(int32(cpuCoreCountRange.Max)),
		MinimumCpuCoreCount: tea.Int32(int32(cpuCoreCountRange.Min)),
		CpuArchitecture:     tea.String("X86"),
	}

	if maxResults != 0 {
		describeInstanceTypesRequest.MaxResults = &maxResults
	}

	// 获取地域下所有满足需要的实例类型
	describeInstanceTypesResponse, err := globals.EcsClient.DescribeInstanceTypesWithContext(ctx, describeInstanceTypesRequest, &dara.RuntimeOptions{})

	if err != nil {
		return nil, err
	}

	var instanceTypeIds = make([]string, 0)
	var instanceTypeInfoMap = make(map[string]*ecs20140526.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType)

	for _, inst := range describeInstanceTypesResponse.Body.InstanceTypes.InstanceType {
		instanceTypeIds = append(instanceTypeIds, *inst.InstanceTypeId)
		instanceTypeInfoMap[*inst.InstanceTypeId] = inst
	}

	var result = make([]InstanceChargeItem, 0)

	var targetZoneItems []globals.ZoneCacheItem

	// 若请求体指定了可用区，则只在该可用区内搜索；否则在当前地域的所有可用区搜索
	if zoneId != "" {
		onlyZone := globals.GetZoneItemByZoneId(zoneId)

		if onlyZone == nil {
			return nil, errors.New("zone id not found")
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
			describePriceResponse, err := globals.EcsClient.DescribePriceWithContext(ctx, describePriceRequest, &dara.RuntimeOptions{})

			currentTypeAndTradePrice := InstanceTypeAndTradePrice{
				InstanceType: instanceType,
			}

			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, err
				}

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

		result = append(result, InstanceChargeItem{
			ZoneId:             *zoneItem.ZoneId,
			TypesAndTradePrice: typesAndTradePrice,
		})
	}

	// 对结果按照指定字段进行排序
	if sortBy != "" {
		for _, item := range result {
			typesAndTradePrice := item.TypesAndTradePrice

			sort.SliceStable(typesAndTradePrice, func(i, j int) bool {
				a := typesAndTradePrice[i]
				b := typesAndTradePrice[j]

				var less bool
				switch sortBy {
				case consts.ICSortByCpuCoreCount:
					if a.CpuCoreCount == b.CpuCoreCount {
						less = false
					} else {
						less = a.CpuCoreCount < b.CpuCoreCount
					}
				case consts.ICSortByMemorySize:
					if a.MemorySize == b.MemorySize {
						less = false
					} else {
						less = a.MemorySize < b.MemorySize
					}
				case consts.ICSortByTradePrice:
					if a.TradePrice == b.TradePrice {
						less = false
					} else {
						less = a.TradePrice < b.TradePrice
					}
				default:
					less = false
				}

				if sortOrder == consts.SortByDesc {
					return !less
				}
				return less
			})
		}
	}

	return result, nil
}

var preferredInstanceChargePresent atomic.Bool
var preferredInstanceCharge InstanceTypeAndTradePrice
var preferredInstanceChargeMu sync.Mutex

func SnapshotPreferredInstanceChargePresent() bool {
	return preferredInstanceChargePresent.Load()
}

func SnapshotPreferredInstanceCharge() InstanceTypeAndTradePrice {
	preferredInstanceChargeMu.Lock()
	defer preferredInstanceChargeMu.Unlock()
	return preferredInstanceCharge
}

func InstanceCharge(quit chan bool) {
	ticker := time.NewTicker(10 * time.Minute)
	logger := log.New(os.Stdout, "[InstanceCharge] ", log.LstdFlags)
	logger.Println("starting...")

	unqualifiedEcsTypeRegex, _ := regexp.Compile(`^ecs\.(e|s6|xn4|n4|mn4|e4|t|d).*$`)

	for {
		func() {
			logger.Println("getting instance charge")

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()

			result, err := GetInstanceCharge(
				ctx,
				*globals.ZoneCache[0].ZoneId,
				IntRange{10, 16},
				IntRange{4, 8},
				consts.ICSortByTradePrice,
				consts.SortByAsc,
				0,
			)

			if err != nil {
				logger.Println("cannot get instance charge", err)
				return
			}

			if len(result) == 0 || len(result[0].TypesAndTradePrice) == 0 {
				logger.Println("got zero length array")
				return
			}

			logger.Println("got instance charge of length", len(result[0].TypesAndTradePrice))

			filtered := make([]InstanceTypeAndTradePrice, 0, len(result[0].TypesAndTradePrice))

			for _, instance := range result[0].TypesAndTradePrice {
				if unqualifiedEcsTypeRegex.Match([]byte(instance.InstanceType)) || instance.TradePrice > 0.6 {
					continue
				}
				filtered = append(filtered, instance)
			}

			logger.Println("filtered length", len(filtered))

			if len(filtered) == 0 {
				logger.Println("warn: no preferred instance found with filter. set to empty.")
				preferredInstanceChargeMu.Lock()
				preferredInstanceCharge = InstanceTypeAndTradePrice{}
				preferredInstanceChargeMu.Unlock()

				if preferredInstanceChargePresent.Load() == true {
					preferredInstanceChargePresent.Store(false)
				}

				logger.Println("next refresh in 5m")
				ticker.Reset(5 * time.Minute)
				return
			}

			if filtered[0].InstanceType != preferredInstanceCharge.InstanceType {
				preferredInstanceChargeMu.Lock()
				preferredInstanceCharge = filtered[0]
				preferredInstanceChargeMu.Unlock()

				if preferredInstanceChargePresent.Load() == false {
					preferredInstanceChargePresent.Store(true)
				}

				logger.Printf("updated preferred instance, new type: %s, new trade price: %.2f, mem: %.2fG, cpu: %d", preferredInstanceCharge.InstanceType, preferredInstanceCharge.TradePrice, preferredInstanceCharge.MemorySize, preferredInstanceCharge.CpuCoreCount)
				logger.Println("next refresh in 5m")
				ticker.Reset(5 * time.Minute)
			} else {
				logger.Println("preferred instance type remains unchanged this time")
				logger.Println("next refresh in 10m")
				ticker.Reset(10 * time.Minute)
			}
		}()

		select {
		case <-ticker.C:
			continue

		case <-quit:
			return
		}
	}
}
