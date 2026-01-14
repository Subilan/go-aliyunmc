package monitors

import (
	"context"
	"testing"

	"github.com/Subilan/go-aliyunmc/clients"
	"github.com/Subilan/go-aliyunmc/config"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
)

type availableInstanceItem struct {
	InstanceType string
	Memory       int
	CpuCoreCount int
	ZoneId       string
	TradePrice   float32
}

func TestInstanceCharge(t *testing.T) {
	config.Load("../config.toml")
	ecsConfig := config.Cfg.GetAliyunEcsConfig()
	regionId := config.Cfg.Aliyun.RegionId

	memChoices := config.Cfg.Monitor.InstanceCharge.MemChoices
	cpuCoreCountChoices := config.Cfg.Monitor.InstanceCharge.CpuCoreCountChoices

	client, err := clients.ShouldCreateEcsClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err != nil {
		t.Fatal(err.Error())
	}

	var result = make([]availableInstanceItem, 0, 10)

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

			describeAvailableResourceResponse, err := client.DescribeAvailableResourceWithContext(ctx, describeAvailableResourceRequest, &dara.RuntimeOptions{})

			if err != nil {
				t.Fatal(err.Error())
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

						describePriceResponse, err := client.DescribePriceWithContext(ctx, describePriceRequest, &dara.RuntimeOptions{})

						if err != nil {
							t.Logf("describe price error: %s", err.Error())
						} else {
							tradePrice = *describePriceResponse.Body.PriceInfo.Price.TradePrice
						}

						result = append(result, availableInstanceItem{
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

	t.Log(result)
}
