// Package scheduler provides error codes for Scheduler Service.
//
// This is an example of how business services should define their error codes
// in their own packages, separate from the core errors package.
package scheduler

import (
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// ============================================================================
// Scheduler Service Configuration
// ============================================================================

// ServiceScheduler is the service code for Scheduler Service.
const ServiceScheduler = 3

func init() {
	errors.RegisterService(ServiceScheduler, "scheduler-service")
}

// ============================================================================
// Scheduler Service Request Errors (Category: 01)
// ============================================================================

var (
	// ErrTaskInvalidCron indicates invalid cron expression.
	ErrTaskInvalidCron = errors.NewRequestErr(ServiceScheduler, 1,
		"Invalid cron expression", "Cron 表达式无效")

	// ErrTaskInvalidParams indicates invalid task parameters.
	ErrTaskInvalidParams = errors.NewRequestErr(ServiceScheduler, 2,
		"Invalid task parameters", "任务参数无效")

	// ErrTaskInvalidName indicates invalid task name.
	ErrTaskInvalidName = errors.NewRequestErr(ServiceScheduler, 3,
		"Invalid task name", "任务名称无效")

	// ErrTaskInvalidHandler indicates invalid task handler.
	ErrTaskInvalidHandler = errors.NewRequestErr(ServiceScheduler, 4,
		"Invalid task handler", "任务处理器无效")

	// ErrTaskInvalidSchedule indicates invalid task schedule.
	ErrTaskInvalidSchedule = errors.NewRequestErr(ServiceScheduler, 5,
		"Invalid task schedule", "任务调度配置无效")
)

// ============================================================================
// Scheduler Service Resource Errors (Category: 04)
// ============================================================================

var (
	// ErrTaskNotFound indicates task not found.
	ErrTaskNotFound = errors.NewNotFoundErr(ServiceScheduler, 1,
		"Task not found", "任务不存在")

	// ErrTaskExecutionNotFound indicates task execution not found.
	ErrTaskExecutionNotFound = errors.NewNotFoundErr(ServiceScheduler, 2,
		"Task execution not found", "任务执行记录不存在")

	// ErrTaskLogNotFound indicates task log not found.
	ErrTaskLogNotFound = errors.NewNotFoundErr(ServiceScheduler, 3,
		"Task log not found", "任务日志不存在")

	// ErrWorkerNotFound indicates worker not found.
	ErrWorkerNotFound = errors.NewNotFoundErr(ServiceScheduler, 4,
		"Worker not found", "工作节点不存在")
)

// ============================================================================
// Scheduler Service Conflict Errors (Category: 05)
// ============================================================================

var (
	// ErrTaskAlreadyExists indicates task already exists.
	ErrTaskAlreadyExists = errors.NewConflictErr(ServiceScheduler, 1,
		"Task already exists", "任务已存在")

	// ErrTaskRunning indicates task is running.
	ErrTaskRunning = errors.NewConflictErr(ServiceScheduler, 2,
		"Task is running", "任务正在执行")

	// ErrTaskDisabled indicates task is disabled.
	ErrTaskDisabled = errors.NewConflictErr(ServiceScheduler, 3,
		"Task is disabled", "任务已禁用")

	// ErrTaskPaused indicates task is paused.
	ErrTaskPaused = errors.NewConflictErr(ServiceScheduler, 4,
		"Task is paused", "任务已暂停")

	// ErrWorkerBusy indicates worker is busy.
	ErrWorkerBusy = errors.NewConflictErr(ServiceScheduler, 5,
		"Worker is busy", "工作节点繁忙")
)

// ============================================================================
// Scheduler Service Internal Errors (Category: 07)
// ============================================================================

var (
	// ErrTaskExecutionFailed indicates task execution failed.
	ErrTaskExecutionFailed = errors.NewInternalErr(ServiceScheduler, 1,
		"Task execution failed", "任务执行失败")

	// ErrTaskSchedulingFailed indicates task scheduling failed.
	ErrTaskSchedulingFailed = errors.NewInternalErr(ServiceScheduler, 2,
		"Task scheduling failed", "任务调度失败")

	// ErrTaskCallbackFailed indicates task callback failed.
	ErrTaskCallbackFailed = errors.NewInternalErr(ServiceScheduler, 3,
		"Task callback failed", "任务回调失败")

	// ErrWorkerRegistrationFailed indicates worker registration failed.
	ErrWorkerRegistrationFailed = errors.NewInternalErr(ServiceScheduler, 4,
		"Worker registration failed", "工作节点注册失败")

	// ErrWorkerHeartbeatFailed indicates worker heartbeat failed.
	ErrWorkerHeartbeatFailed = errors.NewInternalErr(ServiceScheduler, 5,
		"Worker heartbeat failed", "工作节点心跳失败")
)

// ============================================================================
// Scheduler Service Timeout Errors (Category: 11)
// ============================================================================

var (
	// ErrTaskTimeout indicates task execution timeout.
	ErrTaskTimeout = errors.NewTimeoutErr(ServiceScheduler, 1,
		"Task execution timeout", "任务执行超时")

	// ErrWorkerTimeout indicates worker response timeout.
	ErrWorkerTimeout = errors.NewTimeoutErr(ServiceScheduler, 2,
		"Worker response timeout", "工作节点响应超时")
)
