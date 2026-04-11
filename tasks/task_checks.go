package tasks

import (
	"go-aliyunmc/h"
	"go-aliyunmc/states"
	"go-aliyunmc/store"
	"net/http"
	"time"
)

type TaskCheckFunc func(map[string]any) error

var checkMustHaveActiveInstance TaskCheckFunc = func(_ map[string]any) error {
	instance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return err
	}

	if instance == nil {
		return h.HttpError(http.StatusNotFound, "没有可用的实例")
	}

	return nil
}

var checkMustHaveActiveDeployedInstance TaskCheckFunc = func(_ map[string]any) error {
	instance, err := store.GetActiveInstanceDefaultNil()

	if err != nil {
		return err
	}

	if instance == nil {
		return h.HttpError(http.StatusNotFound, "没有可用的实例")
	}

	if !instance.IsDeployed {
		return h.HttpError(http.StatusConflict, "实例未部署")
	}

	return nil
}

var checkMustHaveActiveDeployedRunningInstance TaskCheckFunc = func(_ map[string]any) error {
	if err := checkMustHaveActiveDeployedInstance(nil); err != nil {
		return err
	}

	status, err := states.StableSnapshotInstanceStatus(5 * time.Second)
	
	if err != nil || status.Error != nil {
		return h.HttpError(http.StatusServiceUnavailable, "无法获取最新的实例状态")
	}

	if status.Value != "Running" {
		return h.HttpError(http.StatusServiceUnavailable, "实例未运行")
	}

	return nil
}
