package monitors

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type AvailableInstanceItem struct {
	InstanceType string  `json:"instanceType"`
	Memory       int     `json:"memory"`
	CpuCoreCount int     `json:"cpuCoreCount"`
	ZoneId       string  `json:"zoneId"`
	TradePrice   float32 `json:"tradePrice"`
}

type PreferredInstanceFileContent struct {
	AvailableInstanceItem
	Candidates []AvailableInstanceItem `json:"candidates"`
	UpdatedAt  time.Time               `json:"updatedAt"`
}

// GetInstanceCharge 尝试获取地域下符合要求的所有实例类型，并获取该实例类型在抢占式实例中的每小时预估价格。
func GetInstanceCharge(ctx context.Context, logger *log.Logger) ([]AvailableInstanceItem, error) {
	ecsConfig := config.Cfg.GetAliyunEcsConfig()
	regionId := config.Cfg.Aliyun.RegionId

	memChoices := config.Cfg.Monitor.InstanceCharge.MemChoices
	cpuCoreCountChoices := config.Cfg.Monitor.InstanceCharge.CpuCoreCountChoices

	var result = make([]AvailableInstanceItem, 0, 10)

	for _, mem := range memChoices {
		for _, cpu := range cpuCoreCountChoices {
			describeAvailableResourceRequest := &ecs20140526.DescribeAvailableResourceRequest{
				RegionId:            &regionId,
				InstanceChargeType:  tea.String("PostPaid"),
				SpotStrategy:        tea.String("SpotAsPriceGo"),
				SpotDuration:        tea.Int32(1),
				DestinationResource: tea.String("InstanceType"),
				SystemDiskCategory:  &ecsConfig.SystemDisk.Category,
				DataDiskCategory:    &ecsConfig.DataDisk.Category,
				Cores:               tea.Int32(int32(cpu)),
				Memory:              tea.Float32(float32(mem)),
				ResourceType:        tea.String("instance"),
			}

			describeAvailableResourceResponse, err := clients.EcsClient.DescribeAvailableResourceWithContext(ctx, describeAvailableResourceRequest, &dara.RuntimeOptions{})

			if err != nil {
				return nil, err
			}

			for _, availZone := range describeAvailableResourceResponse.Body.AvailableZones.AvailableZone {
				if len(availZone.AvailableResources.AvailableResource) > 0 {
					resources := availZone.AvailableResources.AvailableResource[0].SupportedResources.SupportedResource

					for _, resource := range resources {
						if *resource.StatusCategory != "WithStock" || *resource.Status != "Available" {
							continue
						}

						var tradePrice float32 = -1

						describePriceRequest := &ecs20140526.DescribePriceRequest{
							RegionId:                &regionId,
							ZoneId:                  availZone.ZoneId,
							ResourceType:            tea.String("instance"),
							InstanceType:            resource.Value,
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
							SpotStrategy: tea.String("SpotAsPriceGo"),
							SpotDuration: tea.Int32(1),
						}

						describePriceResponse, err := clients.EcsClient.DescribePriceWithContext(ctx, describePriceRequest, &dara.RuntimeOptions{})

						if err != nil {
							logger.Println("describe price error: %s", err.Error())
						} else {
							tradePrice = *describePriceResponse.Body.PriceInfo.Price.TradePrice
						}

						filters := config.Cfg.Monitor.InstanceCharge.Filters

						if tradePrice > filters.MaxTradePrice {
							continue
						}

						if filters.InstanceTypeExclusion != "" {
							typeExRegex, err := regexp.Compile(filters.InstanceTypeExclusion)
							if err == nil {
								if typeExRegex.MatchString(*resource.Value) {
									logger.Printf("filtered instance type %s using regex %s", *resource.Value, filters.InstanceTypeExclusion)
									continue
								}
							} else {
								logger.Println("warning: ignored invalid instance type exclusion regular expression: %s", err.Error())
							}
						}

						result = append(result, AvailableInstanceItem{
							ZoneId:       *availZone.ZoneId,
							InstanceType: *resource.Value,
							TradePrice:   tradePrice,
							Memory:       mem,
							CpuCoreCount: cpu,
						})
					}
				}
			}
		}
	}

	// 按照价格排序
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].TradePrice < result[j].TradePrice
	})

	return result, nil
}

var preferredInstanceChargePresent atomic.Bool
var preferredInstanceCharge AvailableInstanceItem
var preferredInstanceChargeMu sync.Mutex
var preferredInstanceChargeCandidates []AvailableInstanceItem
var preferredInstanceChargeCandidatesMu sync.Mutex

// SnapshotPreferredInstanceChargePresent 返回系统是否记录了有效的最佳实例类型及可用区
func SnapshotPreferredInstanceChargePresent() bool {
	return preferredInstanceChargePresent.Load()
}

// SnapshotPreferredInstanceCharge 返回当前获取的最佳实例类型及可用区。在调用此函数之前，除非确信，应当调用 SnapshotPreferredInstanceChargePresent 检查该信息是否存在
func SnapshotPreferredInstanceCharge() AvailableInstanceItem {
	preferredInstanceChargeMu.Lock()
	defer preferredInstanceChargeMu.Unlock()
	return preferredInstanceCharge
}

func InstanceCharge(quit chan bool) {
	cfg := config.Cfg.Monitor.InstanceCharge
	ticker := time.NewTicker(cfg.IntervalDuration())
	logger := log.New(os.Stdout, "[InstanceCharge] ", log.LstdFlags)
	logger.Println("starting...")

	cacheFileContent, err := os.ReadFile(cfg.CacheFile)
	if err != nil {
		logger.Println("read cache file error:", err)
	} else {
		var cacheFileData PreferredInstanceFileContent
		err = json.Unmarshal(cacheFileContent, &cacheFileData)
		if err != nil {
			logger.Println("unmarshal cache file error:", err)
		} else {

			// TODO: add validation for in-file struct
			preferredInstanceChargePresent.Store(true)
			preferredInstanceChargeMu.Lock()
			preferredInstanceCharge = cacheFileData.AvailableInstanceItem
			preferredInstanceChargeMu.Unlock()
			preferredInstanceChargeCandidatesMu.Lock()
			preferredInstanceChargeCandidates = cacheFileData.Candidates
			preferredInstanceChargeCandidatesMu.Unlock()
			logger.Printf("using cache from file, preferred = %v, candidate length = %d", preferredInstanceCharge, len(preferredInstanceChargeCandidates))
		}
	}

	for {
		func() {
			logger.Println("getting instance charge")

			ctx, cancel := context.WithTimeout(context.Background(), cfg.TimeoutDuration())
			defer cancel()

			result, err := GetInstanceCharge(ctx, logger)

			if err != nil {
				logger.Println("cannot get instance charge", err)
				return
			}

			if len(result) == 0 {
				logger.Println("warn: no preferred instance found with filter. set to empty.")
				preferredInstanceChargeMu.Lock()
				preferredInstanceCharge = AvailableInstanceItem{}
				preferredInstanceChargeMu.Unlock()

				if preferredInstanceChargePresent.Load() == true {
					preferredInstanceChargePresent.Store(false)
				}

				logger.Println("next refresh in", cfg.RetryIntervalDuration())
				ticker.Reset(cfg.RetryIntervalDuration())
				return
			}

			target := result[0]
			candidates := make([]AvailableInstanceItem, 0, 3)

			for i := 1; i < min(len(result)-1, 3); i++ {
				candidates = append(candidates, result[i])
			}

			if target.InstanceType != preferredInstanceCharge.InstanceType {
				preferredInstanceChargeMu.Lock()
				preferredInstanceCharge = target
				preferredInstanceChargeMu.Unlock()
				preferredInstanceChargeCandidatesMu.Lock()
				preferredInstanceChargeCandidates = candidates
				preferredInstanceChargeCandidatesMu.Unlock()

				if preferredInstanceChargePresent.Load() == false {
					preferredInstanceChargePresent.Store(true)
				}

				logger.Printf("updated preferred instance, %v", target)

				marshalled, _ := json.Marshal(PreferredInstanceFileContent{AvailableInstanceItem: target, Candidates: candidates, UpdatedAt: time.Now()})
				err = os.WriteFile(cfg.CacheFile, marshalled, 0600)

				if err != nil {
					logger.Println("cannot write to cache file", err)
				} else {
					logger.Println("updated cache file")
				}

				logger.Println("next refresh in", cfg.RetryIntervalDuration())
				ticker.Reset(cfg.RetryIntervalDuration())
			} else {
				logger.Println("preferred instance type remains unchanged this time")
				logger.Println("next refresh in", cfg.IntervalDuration())
				ticker.Reset(cfg.IntervalDuration())
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
