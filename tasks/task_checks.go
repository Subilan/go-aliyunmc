package tasks

import (
	"github.com/Subilan/go-aliyunmc/h"
	"github.com/Subilan/go-aliyunmc/states"
	"github.com/Subilan/go-aliyunmc/store"
	"net/http"
	"time"
)

// 来自 monitors 包的函数，用于避免 import cycle
var (
	SnapshotServerStatus     func() states.State[states.ServerStatusState]
	SnapshotBestEcsCandidate func() states.State[states.EcsCandidate]
	WaitInstanceSnapshot     func(time.Duration) (states.State[string], error)
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

	snap, err := WaitInstanceSnapshot(5 * time.Second)

	if err != nil || !snap.IsValid() {
		return h.HttpError(http.StatusServiceUnavailable, "无法获取最新的实例状态")
	}
	if snap.Value != "Running" {
		return h.HttpError(http.StatusServiceUnavailable, "实例未运行")
	}

	return nil
}
