package runtime

import (
	"time"

	"neo-code/internal/subagent"
)

// EventPermissionRequest 为兼容旧事件名保留，语义等同 EventPermissionRequested。
const EventPermissionRequest EventType = EventPermissionRequested

// EventCompactDone 为兼容旧事件名保留，语义等同 EventCompactApplied。
const EventCompactDone EventType = EventCompactApplied

// SubAgentEventPayload 描述子代理执行生命周期的事件载荷。
type SubAgentEventPayload struct {
	Role                subagent.Role       `json:"role"`
	Executor            string              `json:"executor,omitempty"`
	TaskID              string              `json:"task_id"`
	State               subagent.State      `json:"state"`
	StopReason          subagent.StopReason `json:"stop_reason,omitempty"`
	Step                int                 `json:"step,omitempty"`
	Attempts            int                 `json:"attempts,omitempty"`
	StartedAt           time.Time           `json:"started_at,omitempty"`
	EndedAt             time.Time           `json:"ended_at,omitempty"`
	QueueSize           int                 `json:"queue_size,omitempty"`
	Running             int                 `json:"running,omitempty"`
	DispatchConcurrency int                 `json:"dispatch_concurrency,omitempty"`
	PendingApproval     bool                `json:"pending_approval,omitempty"`
	Reason              string              `json:"reason,omitempty"`
	Delta               string              `json:"delta,omitempty"`
	Error               string              `json:"error,omitempty"`
}

// SubAgentToolCallEventPayload 描述子代理工具调用事件载荷。
type SubAgentToolCallEventPayload struct {
	Role      subagent.Role `json:"role"`
	TaskID    string        `json:"task_id"`
	ToolName  string        `json:"tool_name"`
	Decision  string        `json:"decision"`
	ElapsedMS int64         `json:"elapsed_ms"`
	Truncated bool          `json:"truncated"`
	Error     string        `json:"error,omitempty"`
}

const (
	// EventSubAgentTaskStarted 在子代理任务启动后触发。
	EventSubAgentTaskStarted EventType = "subagent_task_started"
	// EventSubAgentTaskProgress 在子代理执行每一步后触发。
	EventSubAgentTaskProgress EventType = "subagent_task_progress"
	// EventSubAgentTaskRetried 在子代理任务进入重试后触发。
	EventSubAgentTaskRetried EventType = "subagent_task_retried"
	// EventSubAgentTaskBlocked 在子代理任务被阻塞（依赖或退避）时触发。
	EventSubAgentTaskBlocked EventType = "subagent_task_blocked"
	// EventSubAgentTaskCompleted 在子代理成功结束后触发。
	EventSubAgentTaskCompleted EventType = "subagent_task_completed"
	// EventSubAgentTaskFailed 在子代理失败结束后触发。
	EventSubAgentTaskFailed EventType = "subagent_task_failed"
	// EventSubAgentDispatchTaskFailed 在调度层判定任务失败（如依赖失败）时触发。
	EventSubAgentDispatchTaskFailed EventType = "subagent_dispatch_task_failed"
	// EventSubAgentTaskCanceled 在子代理被取消后触发。
	EventSubAgentTaskCanceled EventType = "subagent_task_canceled"
	// EventSubAgentDispatchFinished 在一次调度轮次结束后触发。
	EventSubAgentDispatchFinished EventType = "subagent_dispatch_finished"

	// EventSubAgentStarted 为旧常量别名，语义等同 EventSubAgentTaskStarted。
	EventSubAgentStarted EventType = EventSubAgentTaskStarted
	// EventSubAgentProgress 为旧常量别名，语义等同 EventSubAgentTaskProgress。
	EventSubAgentProgress EventType = EventSubAgentTaskProgress
	// EventSubAgentRetried 为旧常量别名，语义等同 EventSubAgentTaskRetried。
	EventSubAgentRetried EventType = EventSubAgentTaskRetried
	// EventSubAgentBlocked 为旧常量别名，语义等同 EventSubAgentTaskBlocked。
	EventSubAgentBlocked EventType = EventSubAgentTaskBlocked
	// EventSubAgentCompleted 为旧常量别名，语义等同 EventSubAgentTaskCompleted。
	EventSubAgentCompleted EventType = EventSubAgentTaskCompleted
	// EventSubAgentFailed 为旧常量别名，语义等同 EventSubAgentTaskFailed。
	EventSubAgentFailed EventType = EventSubAgentTaskFailed
	// EventSubAgentCanceled 为旧常量别名，语义等同 EventSubAgentTaskCanceled。
	EventSubAgentCanceled EventType = EventSubAgentTaskCanceled
	// EventSubAgentFinished 为旧常量别名，语义等同 EventSubAgentDispatchFinished。
	EventSubAgentFinished EventType = EventSubAgentDispatchFinished
	// EventSubAgentToolCallStarted 在子代理发起工具调用时触发。
	EventSubAgentToolCallStarted EventType = "subagent_tool_call_started"
	// EventSubAgentToolCallResult 在子代理工具调用返回后触发。
	EventSubAgentToolCallResult EventType = "subagent_tool_call_result"
	// EventSubAgentToolCallDenied 在子代理工具调用被权限拒绝时触发。
	EventSubAgentToolCallDenied EventType = "subagent_tool_call_denied"
)
