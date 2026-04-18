package global_states

import "sync/atomic"

// archiving 是一个用来表示是否有正在执行的归档流程的全局状态变量。
// 设置这一变量而不是直接使用 tasks.IsExclusiveTaskRunning 的原因是归档流程不一定是由任务触发的，例如自动回收流程也会设置这个状态。
// 通过这个变量，监控器或者其他流程可以统一地检查是否有归档流程在执行，而不需要关心具体是哪个模块在执行归档流程。
var archiving atomic.Bool

// IsArchiving 返回当前是否有正在执行的归档流程。关于该变量的具体含义，请参考 archiveTaskRunning 的定义。
func IsArchiving() bool {
	return archiving.Load()
}

// SetArchiving 设置当前归档流程的运行状态。
func SetArchiving(running bool) {
	archiving.Store(running)
}