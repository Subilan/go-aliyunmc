package tasks

import (
	"context"
	"errors"
	"fmt"
	"go-aliyunmc/aliyun"
	"go-aliyunmc/h"
	"go-aliyunmc/log_util"
	"go-aliyunmc/remote_util"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
	"go-aliyunmc/store/models"
	"net/http"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
)

// CreateInstanceTaskArgs 是创建实例任务的参数结构体。
type CreateInstanceTaskArgs struct {
	// InstanceType 是实例的规格
	InstanceType string `json:"instanceType"`
	// ZoneId 是实例的可用区ID
	ZoneId string `json:"zoneId"`
	// VSwitchId 是实例的交换机ID，如果 UseDefaultVSwitch 为true则可以不填
	VSwitchId string `json:"vSwitchId"`
	// UseDefaultVSwitch 指定是否使用默认交换机，如果为true则会自动查询或创建指定可用区的默认交换机
	UseDefaultVSwitch bool `json:"useDefaultVSwitch"`
	// StartWhenCreated 指定实例创建成功后是否立即尝试启动，如果为true则会在创建并分配IP后触发启动操作并等待实例SSH可用
	StartWhenCreated bool `json:"startWhenCreated"`
}

// deleteInstance 用于清除创建实例过程中产生的资源，避免残留未使用的实例占用配额或产生费用。
// 它会尝试删除指定ID的实例，如果删除失败会记录错误日志但不会返回错误。
func deleteInstance(id string) {
	deleteInstanceRequest := &ecs20140526.DeleteInstanceRequest{
		InstanceId: tea.String(id),
	}

	_, err := aliyun.EcsClient.DeleteInstance(deleteInstanceRequest)
	if err != nil {
		log_util.Error("在清理过程中无法删除实例：%v", err)
	}
}

// startInstanceWithRetry 尝试启动实例，如果启动失败会每隔5秒重试一次，直到成功或达到1分钟的超时限制。
// 设置重试机制的原因是，实例被创建之后有一定概率（较小）长时间处于Initializing状态，该状态不支持启动操作。
func startInstanceWithRetry(startInstanceRequest *ecs20140526.StartInstanceRequest, tc *TaskContext) error {
	startCtx, startCancel := context.WithTimeout(tc.Context(), 1*time.Minute)
	defer startCancel()
	startRetryTicker := time.NewTicker(5 * time.Second)
	defer startRetryTicker.Stop()

	var startErr error

	for {
		_, startErr = aliyun.EcsClient.StartInstance(startInstanceRequest)
		if startErr == nil {
			break
		}

		tc.println("开启实例失败，5秒后重试")

		select {
		case <-startCtx.Done():
			if errors.Is(startCtx.Err(), context.Canceled) {
				tc.println("尝试开启实例过程中任务被取消")
				return fmt.Errorf("任务被取消")
			}
			return fmt.Errorf("开启实例超时(60s): %w，请尝试手动开启", startErr)
		case <-startRetryTicker.C:
		}
	}

	return nil
}

// getDefaultVSwitchId 尝试获取指定可用区的默认交换机ID，如果没有找到则创建一个新的默认交换机并等待其就绪后返回ID。
func getDefaultVSwitchId(tc *TaskContext, zoneId string) (string, error) {
	tc.println("开始查询默认交换机")
	describeVSwitchesRequest := &vpc20160428.DescribeVSwitchesRequest{
		ZoneId:    tea.String(zoneId),
		IsDefault: tea.Bool(true),
	}
	describeVSwitchesResponse, err := aliyun.VpcClient.DescribeVSwitches(describeVSwitchesRequest)
	if err != nil {
		tc.println("获取默认交换机失败：" + err.Error())
		return "", fmt.Errorf("获取默认VSwitch失败: %w", err)
	}

	if len(describeVSwitchesResponse.Body.VSwitches.VSwitch) > 0 {
		vswitchId := describeVSwitchesResponse.Body.VSwitches.VSwitch[0].VSwitchId
		tc.println("成功查询到现有的默认交换机：" + *vswitchId)
		return tea.StringValue(vswitchId), nil
	}

	tc.println("找不到默认交换机，创建")

	createDefaultVSwitchRequest := &vpc20160428.CreateDefaultVSwitchRequest{
		ZoneId:   tea.String(zoneId),
		RegionId: tea.String(aliyun.C.RegionId),
	}
	createDefaultVSwitchResponse, err := aliyun.VpcClient.CreateDefaultVSwitch(createDefaultVSwitchRequest)

	if err != nil {
		tc.println("创建默认交换机失败：" + err.Error())
		return "", fmt.Errorf("创建默认VSwitch失败: %w", err)
	}

	vswitchId := tea.StringValue(createDefaultVSwitchResponse.Body.VSwitchId)
	tc.println("成功创建默认交换机：" + vswitchId)
	tc.println("等待交换机准备就绪")
	waitVpcReadyCtx, cancel := context.WithTimeout(tc.Context(), 1*time.Minute)
	defer cancel()
	waitVpcReadyTicker := time.NewTicker(5 * time.Second)
	defer waitVpcReadyTicker.Stop()

	for {
		select {
		case <-waitVpcReadyCtx.Done():
			tc.println("任务被取消或交换机状态检查超时")
			if errors.Is(waitVpcReadyCtx.Err(), context.Canceled) {
				return "", fmt.Errorf("任务被取消")
			}
			return "", fmt.Errorf("等待默认VSwitch可用超时")
		case <-waitVpcReadyTicker.C:
			describeVSwitchesResponse, err := aliyun.VpcClient.DescribeVSwitches(describeVSwitchesRequest)
			if err != nil {
				continue
			}
			if len(describeVSwitchesResponse.Body.VSwitches.VSwitch) > 0 {
				if tea.StringValue(describeVSwitchesResponse.Body.VSwitches.VSwitch[0].Status) != "Available" {
					tc.println("交换机未就绪...持续检查中")
					continue
				}
				tc.println("默认交换机已就绪，创建完毕")
				return vswitchId, nil
			}
		}
	}
}

func createInstanceTask(tc *TaskContext, args map[string]any) error {
	var params CreateInstanceTaskArgs

	if err := ShouldBindArgs(args, &params); err != nil {
		return err
	}

	if params.InstanceType == "" || params.ZoneId == "" {
		candidate, ok := states.SnapshotBestEcsCandidate()
		if !ok {
			return fmt.Errorf("未提供实例类型且无法获取最佳实例")
		}
		params.InstanceType = candidate.InstanceType
		params.ZoneId = candidate.ZoneId
	}

	tc.nextStep()
	tc.println("检查当前实例")

	instance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return err
	}

	if instance != nil {
		tc.println("已存在活跃实例")
		return fmt.Errorf("已存在活跃实例")
	}

	tc.nextStep()
	vswitchId := params.VSwitchId
	if params.UseDefaultVSwitch {
		vswitchId, err = getDefaultVSwitchId(tc, params.ZoneId)
		if err != nil {
			tc.println("实例创建失败：无法获取默认交换机")
			return err
		}
		tc.nextStep()
	} else if vswitchId == "" {
		return fmt.Errorf("未指定交换机ID")
	}
	tc.println("开始创建实例")

	ecsConfig := aliyun.C.Ecs

	createInstanceRequest := &ecs20140526.CreateInstanceRequest{
		RegionId:     tea.String(aliyun.C.RegionId),
		ZoneId:       tea.String(params.ZoneId),
		InstanceType: tea.String(params.InstanceType),
		SystemDisk: &ecs20140526.CreateInstanceRequestSystemDisk{
			Category: tea.String(ecsConfig.SystemDisk.Category),
			Size:     tea.Int32(int32(ecsConfig.SystemDisk.Size)),
		},
		DataDisk: []*ecs20140526.CreateInstanceRequestDataDisk{
			{
				Category: tea.String(ecsConfig.DataDisk.Category),
				Size:     tea.Int32(int32(ecsConfig.DataDisk.Size)),
				DiskName: tea.String("data"),
			},
		},
		InternetChargeType:       tea.String("PayByTraffic"), // This line costs CNY 400.
		InternetMaxBandwidthOut:  tea.Int32(int32(ecsConfig.InternetMaxBandwidthOut)),
		HostName:                 tea.String(ecsConfig.HostName),
		Password:                 tea.String(ecsConfig.RootPassword),
		InstanceChargeType:       tea.String("PostPaid"),
		SpotStrategy:             tea.String("SpotAsPriceGo"),
		SpotDuration:             tea.Int32(1),
		SpotInterruptionBehavior: tea.String(ecsConfig.SpotInterruptionBehavior),
		SecurityGroupId:          tea.String(ecsConfig.SecurityGroupId),
		VSwitchId:                tea.String(vswitchId),
		ImageId:                  tea.String(ecsConfig.ImageId),
	}

	createInstanceResponse, err := aliyun.EcsClient.CreateInstance(createInstanceRequest)

	if err != nil {
		return fmt.Errorf("创建实例失败: %w", err)
	}

	instance = &models.Instance{
		InstanceId:   tea.StringValue(createInstanceResponse.Body.InstanceId),
		InstanceType: params.InstanceType,
		ZoneId:       params.ZoneId,
		VSwitchId:    vswitchId,
		RegionId:     aliyun.C.RegionId,
	}

	if err := store.DB.Save(instance).Error; err != nil {
		deleteInstance(instance.InstanceId)
		return fmt.Errorf("保存实例信息失败: %w", err)
	}

	tc.println("实例创建成功！")
	tc.nextStep()

	tc.println("开始分配IP地址")

	allocatePublicIpAddressRequest := &ecs20140526.AllocatePublicIpAddressRequest{
		InstanceId: &instance.InstanceId,
	}
	allocatePublicIpAddressResponse, err := aliyun.EcsClient.AllocatePublicIpAddress(allocatePublicIpAddressRequest)

	if err != nil {
		deleteInstance(instance.InstanceId)
		return fmt.Errorf("分配IP地址失败: %w", err)
	}

	instance.Ip = tea.StringValue(allocatePublicIpAddressResponse.Body.IpAddress)

	if err := store.DB.Save(instance).Error; err != nil {
		deleteInstance(instance.InstanceId)
		return fmt.Errorf("保存实例IP信息失败: %w", err)
	}

	tc.println("IP地址分配成功：" + instance.Ip)

	if !params.StartWhenCreated {
		return nil
	}

	tc.nextStep()
	tc.println("尝试触发实例启动")
	startInstanceRequest := &ecs20140526.StartInstanceRequest{
		InstanceId: &instance.InstanceId,
	}
	if err := startInstanceWithRetry(startInstanceRequest, tc); err != nil {
		deleteInstance(instance.InstanceId)
		return err
	}
	tc.println("已触发实例启动")
	tc.nextStep()
	tc.println("开始网络验证")

	waitSSHCtx, waitSSHCancel := context.WithTimeout(tc.Context(), 1*time.Minute)
	defer waitSSHCancel()
	waitSSHTicker := time.NewTicker(3 * time.Second)
	defer waitSSHTicker.Stop()

outer:
	for {
		select {
		case <-waitSSHCtx.Done():
			tc.println("任务被取消或网络验证超时")
			if errors.Is(waitSSHCtx.Err(), context.Canceled) {
				return fmt.Errorf("任务被取消")
			}
			return fmt.Errorf("等待实例SSH可用超时")
		case <-waitSSHTicker.C:
			tc.println("尝试连接...")
			if remote_util.TryDialRoot(instance.Ip, 5*time.Second) {
				tc.println("连接成功")
				break outer
			}
		}
	}

	tc.println("实例已开启且可连接")

	return nil
}

func checkCreateInstanceTask(args map[string]any) error {
	activeInstance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return err
	}

	if activeInstance != nil {
		return h.HttpError(http.StatusConflict, "实例已存在")
	}

	return nil
}

func enforceCreateInstanceTask(role string, args map[string]any) error {
	return enforceRuleIfArgsMatch(
		role,
		"create-custom-instance",
		args,
		func(param CreateInstanceTaskArgs) bool {
			return param.ZoneId != "" || 
			param.InstanceType != "" || 
			param.StartWhenCreated == false || 
			param.VSwitchId != "" || 
			param.UseDefaultVSwitch == false
		},
	)
}
