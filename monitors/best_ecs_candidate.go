package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/env"
	"go-aliyunmc/global_states"
	"go-aliyunmc/log_util"
	"go-aliyunmc/states"
	"os"
	"regexp"
	"sort"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type BestEcsCandidateMonitor struct {
	store    *states.HubbedStore[states.EcsCandidate]
	logger   *log_util.NamedLogger
	interval time.Duration
}

func newBestEcsCandidateMonitor() *BestEcsCandidateMonitor {
	return &BestEcsCandidateMonitor{
		store:    states.NewRecordedHubbedStore[states.EcsCandidate](states.HSKeyBestEcsCandidate),
		logger:   log_util.NewNamedLogger("[monitor/ecs-candidate] ", "best-ecs-candidate-monitor"),
		interval: time.Duration(BestEcsCandidateC.PollIntervalSec) * time.Second,
	}
}

func (m *BestEcsCandidateMonitor) run(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	cacheFileContent, err := os.ReadFile(BestEcsCandidateC.CacheFile)
	if err == nil {
		var cacheFileData states.EcsCandidate
		err = json.Unmarshal(cacheFileContent, &cacheFileData)
		if err != nil {
			m.logger.Error("无法反序列化缓存数据：%s", err.Error())
		} else {
			m.logger.Info("已读取缓存文件")
			m.store.Store(cacheFileData, m.logger)
		}
	} else {
		if !os.IsNotExist(err) {
			m.logger.Error("无法读取缓存文件：%s", err.Error())
		}
	}

	m.pollAndStore(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.pollAndStore(ctx)
		}
	}
}

func (m *BestEcsCandidateMonitor) pollAndStore(ctx context.Context) {
	m.logger.Info("开始查询实例价格")
	candidates, err := m.getInstanceCharge(ctx)
	if err != nil {
		m.store.StoreError(err, m.logger)
		m.logger.Error("failed to get instance charge, skipping: %s", err.Error())
		return
	}

	if len(candidates) > 0 {
		bestCandidate := candidates[0]
		global_states.SetCurrentEcsCandidates(candidates)
		changed := m.store.Store(bestCandidate, m.logger)

		if changed {
			cacheData, err := json.Marshal(bestCandidate)
			if err != nil {
				m.logger.Error("无法序列化缓存数据：%s", err.Error())
			} else {
				err = os.WriteFile(BestEcsCandidateC.CacheFile, cacheData, 0644)
				if err != nil {
					m.logger.Error("无法写入缓存数据：%s", err.Error())
				} else {
					m.logger.Info("已更新缓存数据：%s", bestCandidate.String())
				}
			}
		} else {
			m.logger.Info("最优实例未改变")
		}
	} else {
		m.store.StoreError(fmt.Errorf("找不到可用的 ECS 候选实例"), m.logger)
		m.logger.Info("no available ecs candidate found")
	}
}

// getInstanceCharge 调用阿里云 ECS API 获取符合配置要求的实例规格列表，并按照价格升序排序返回
func (m *BestEcsCandidateMonitor) getInstanceCharge(ctx context.Context) ([]states.EcsCandidate, error) {
	ecsConfig := aliyun.C.Ecs
	regionId := aliyun.C.RegionId

	memChoices := BestEcsCandidateC.MemChoices
	cpuCoreCountChoices := BestEcsCandidateC.CpuCoreCountChoices

	typeExRegex, regexErr := regexp.Compile(BestEcsCandidateC.Filters.InstanceTypeExclusion)

	var result = make([]states.EcsCandidate, 0, 10)

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

			describeAvailableResourceResponse, err := aliyun.EcsClient.DescribeAvailableResourceWithContext(ctx, describeAvailableResourceRequest, &dara.RuntimeOptions{})

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

						describePriceResponse, err := aliyun.EcsClient.DescribePriceWithContext(ctx, describePriceRequest, &dara.RuntimeOptions{})

						if err != nil {
							m.logger.Error("describe price error: %s", err.Error())
						} else {
							tradePrice = *describePriceResponse.Body.PriceInfo.Price.TradePrice
						}

						filters := BestEcsCandidateC.Filters

						if tradePrice > filters.MaxTradePrice {
							continue
						}

						if filters.InstanceTypeExclusion != "" {
							if regexErr == nil {
								if typeExRegex.MatchString(*resource.Value) {
									if env.DEV {
										m.logger.Dev("filtered instance type %s using regex %s", *resource.Value, filters.InstanceTypeExclusion)
									}
									continue
								}
							} else {
								m.logger.Warn("ignored invalid instance type exclusion regular expression: %s", regexErr.Error())
							}
						}

						result = append(result, states.EcsCandidate{
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

	// 按照价格升序排序
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].TradePrice < result[j].TradePrice
	})

	return result, nil
}
