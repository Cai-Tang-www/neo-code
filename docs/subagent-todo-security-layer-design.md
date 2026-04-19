# Tool / SubAgent / Todo 安全层总设计文档

## 1. 文档目标与范围

本文档给出 NeoCode 当前版本中 `tool -> runtime -> subagent -> todo` 的安全层总设计，覆盖以下内容：

- 权限与沙箱的接口边界
- `todo_write` 与 `session todo` 的状态机约束
- subagent 调度链路（含 mixed executor）
- 事件语义与可观测性
- 失败恢复、重试与收敛机制
- 后续扩展时必须保持的安全不变量

不覆盖 UI 视觉实现与 provider 厂商协议细节。

## 2. 分层边界（必须遵守）

主链路：`TUI -> Runtime -> Tools Manager -> Security -> Tool Executor -> Session Store`

子代理链路：`Runtime.dispatchTodos -> SubAgent Scheduler -> RunSubAgentTask -> RuntimeSubAgentEngine -> Tool Executor`

职责边界：

- `internal/tui`：只展示事件与承接用户审批，不直接执行工具。
- `internal/runtime`：ReAct 主循环、工具执行编排、权限事件桥接、subagent 调度入口。
- `internal/tools`：统一工具契约、权限动作映射、会话级审批记忆、执行器分发。
- `internal/security`：权限决策（allow/deny/ask）、workspace 沙箱、capability 校验。
- `internal/session`：Todo 领域模型、状态机、revision 并发控制、持久化一致性。
- `internal/subagent`：DAG 调度与 Worker 生命周期，不直接依赖 TUI。

## 3. 核心数据模型

### 3.1 Todo 模型（`internal/session/todo.go`）

`TodoItem` 关键字段：

- 基础字段：`id`, `content`, `status`, `dependencies`, `priority`, `executor`
- 归属字段：`owner_type`, `owner_id`
- 产物字段：`acceptance`, `artifacts`, `failure_reason`
- 重试字段：`retry_count`, `retry_limit`, `next_retry_at`
- 并发字段：`revision`
- 审计字段：`created_at`, `updated_at`

状态枚举：

- `pending`
- `in_progress`
- `blocked`
- `completed`
- `failed`
- `canceled`

执行器枚举：

- `agent`
- `subagent`

### 3.2 Todo 状态迁移规则

合法迁移（`ValidTransition`）：

- `pending -> in_progress | blocked | failed | canceled`
- `in_progress -> completed | failed | blocked | canceled`
- `blocked -> pending | in_progress | failed | canceled`
- 同态迁移允许（`from == to`）

不变量：

- 终态：`completed/failed/canceled`
- 依赖检查：目标变为 `in_progress/completed` 时，所有依赖必须 `completed`
- 图约束：拒绝自依赖、拒绝未知依赖、拒绝依赖环
- 并发控制：`expected_revision` 不匹配会触发 `ErrRevisionConflict`

## 4. 接口总览

### 4.1 Tool 统一接口（`internal/tools/types.go`）

- `Tool`：`Name/Description/Schema/MicroCompactPolicy/Execute`
- `ToolCallInput`：包含 `SessionID/TaskID/AgentID/CapabilityToken/WorkspacePlan/SessionMutator`
- `ToolResult`：`ToolCallID/Name/Content/IsError/Metadata`

### 4.2 Todo 写接口（SessionMutator）

工具可调用的 Todo 能力：

- `ReplaceTodos`, `AddTodo`, `UpdateTodo`, `SetTodoStatus`
- `RetryTodo`, `DeleteTodo`
- `ClaimTodo`, `CompleteTodo`, `FailTodo`

Runtime 通过 `runtimeSessionMutator` 保证“内存改动 + 落盘 + 状态替换”的原子过程。

### 4.3 SubAgent 调度接口（`internal/subagent/scheduler_types.go`）

- `TodoStore`：调度器对 Todo 的最小读写依赖
- `Factory` / `WorkerRuntime`：创建并运行子代理 worker
- `SchedulerConfig`：并发、重试、恢复、上下文切片、事件观察器

### 4.4 权限与沙箱接口

- `security.PermissionEngine`：`Check(ctx, action) -> CheckResult`
- `tools.DefaultManager`：工具执行前统一做权限 + capability + sandbox
- `security.WorkspaceSandbox`：路径边界验证（含 symlink 逃逸检查与执行前二次校验）

## 5. todo_write 协议与安全约束

### 5.1 Action 协议

`todo_write` 支持动作：

- `plan`
- `add`
- `update`
- `set_status`
- `remove`
- `claim`
- `complete`
- `fail`
- `retry`

### 5.3 spawn_subagent 协议（仅创建，不执行）

`spawn_subagent` 作为动态 DAG 入口，仅允许创建 `executor=subagent` 的 Todo 项，不直接触发执行。

约束：

- 输入字段：`items[].id/content/dependencies/priority/acceptance/retry_limit`
- 创建时强制：`status=pending`、`executor=subagent`
- 拒绝重复 ID / 未知依赖 / 自依赖 / 依赖环
- 保持与 `todo_write` 一致的输入风控上限（体积、数量、字符串长度）

调度仍由 `runtime.dispatchTodos -> subagent scheduler` 主链路统一推进，避免工具层绕过调度策略。

### 5.2 Schema 一致性

`Schema()` 与解析/校验逻辑对齐点：

- `item.executor` 与 `patch.executor` 都显式枚举 `agent|subagent`
- `item` 兼容 legacy `title` 字段（映射到 `content`）
- `expected_revision` 支持乐观并发

### 5.4 输入风控（`internal/tools/todo/common.go`）

- 参数体积上限：`64KB`
- Todo 项数量上限：`64`
- 字符串字段长度上限：`1024`
- 字符串列表项上限：`64`

错误分类映射：

- `invalid_arguments`
- `invalid_action`
- `todo_not_found`
- `invalid_transition`
- `dependency_violation`
- `revision_conflict`

## 6. 权限审批与会话记忆

### 6.1 工具执行前置链路

`tools.DefaultManager.Execute` 顺序：

1. `buildPermissionAction`（工具参数 -> 安全动作）
2. `verifyCapabilityToken`（子代理令牌签名/绑定/时效校验）
3. `PermissionEngine.Check`
4. 若命中 `ask`，尝试 `sessionPermissionMemory` 自动决议
5. 若允许，进入 `WorkspaceSandbox.Check`
6. 通过后才执行真实工具

### 6.2 ask 决议链路（Runtime）

`executeToolCallWithPermission` 在 `ask` 场景：

1. 发出 `permission_requested`
2. 通过 `approval.Broker` 等待 UI 决议
3. 收到 `allow_once/allow_session/reject`
4. 发出 `permission_resolved`
5. 若允许，记忆会话权限并重试同一工具调用

注意：审批等待不受工具超时约束，避免“用户未点审批”被误判为工具失败。

补充：当等待阶段命中运行上下文 deadline（例如 subagent 预算超时），runtime 会返回
`approval_pending` 语义，调度器将任务收敛为 `blocked`，而不是直接终态 `failed`。

### 6.3 session 级权限记忆（防重复 ask）

记忆维度键：`action_type | category | target_scope`

关键归一化：

- URL：归一到 `host[:port]`
- 路径：`Clean + ToSlash + lower`
- 命令：折叠空白/换行并小写
- MCP：归一到 `mcp.<server>`

scope：

- `once`
- `always_session`
- `reject`

## 7. SubAgent 调度与 Todo 回写

### 7.1 Runtime 调度入口（`dispatchTodos`）

触发时机：

- assistant 本轮无 `tool_calls` 时
- assistant 的 `tool_calls` 执行完成后

运行配置（当前默认）：

- 并发：`2`
- 任务超时：`5m`
- 最大重试：`2`
- `DispatchOnce = true`（单轮调度，避免轮询卡住主循环）

返回 `progressed=true` 条件：

- 本轮有 `succeeded/failed/recovered/retried` 任一变化
- 或存在 `subagent` 任务在等待 `agent` 依赖补齐

### 7.2 Scheduler 行为

- 只调度 `executor=subagent` 且非终态任务
- 依赖未满足：收敛为 `blocked`
- 依赖已失败/取消：下游直接 `failed`，原因 `dependency_failed:*`
- 领取任务：`ClaimTodo -> owner_type=subagent, owner_id=<worker-id>`
- 成功：`CompleteTodo` 并写入 artifacts
- 失败：按 `retry_limit/max_retries` 进入 `blocked(backoff)` 或 `failed`
- 审批等待：统一收敛为 `blocked + reason=approval_pending`
- 依赖失败传导：上游 `failed/canceled` 时，下游直接 `failed(reason=dependency_failed:...)`

### 7.3 subagent worker 执行链

`runtimeSchedulerWorker.Step` -> `RunSubAgentTask` -> `runtimeSubAgentEngine.RunStep`

单步内闭环：

1. provider 生成 assistant
2. 若有工具调用，走 runtime 统一工具执行（含权限和沙箱）
3. tool result 回灌模型
4. 输出满足结构化契约后结束

输出契约必需键：

- `summary`
- `findings`
- `patches`
- `risks`
- `next_actions`
- `artifacts`

## 8. 事件语义（安全层重点）

### 8.1 权限事件

- `permission_requested`
- `permission_resolved`

用于审计字段：`tool/action/target/rule_id/decision/remember_scope/resolved_as`

### 8.2 Todo 事件

- `todo_updated`：`todo_write` 成功
- `todo_conflict`：`revision/dependency/transition` 等冲突类失败

### 8.3 SubAgent 事件

生命周期：

- `subagent_task_started`
- `subagent_task_progress`
- `subagent_task_retried`
- `subagent_task_blocked`
- `subagent_task_completed`
- `subagent_task_failed`
- `subagent_task_canceled`

调度器专属：

- `subagent_dispatch_finished`

`subagent_dispatch_finished` 语义：

- 表示“调度轮次结束事实事件”，不是单任务状态
- `reason` 固定：`dispatch_round_finished`
- 携带：`queue_size`, `running`, `dispatch_concurrency`

`subagent_task_blocked` 在 `reason=approval_pending` 时附带 `pending_approval=true`，用于 UI 和诊断区分“依赖阻塞”与“审批等待”。

### 8.4 事件顺序与可校验性

所有 runtime 事件都带：

- `sequence`：单次 run 内严格递增
- `timestamp`：事件生成时间
- `turn/phase`：运行轮次和阶段

消费端可据此检测乱序、重复与缺失。

## 9. 收敛与防死循环机制

Runtime 双保险：

- `ErrRepeatCycleLimit`：连续重复同工具同参数
- `ErrNoProgressStreakLimit`：连续无实质进展（含 dispatch 前后 Todo 状态签名不变）

作用：

- 防止 provider 持续“空转回复”导致无限循环
- 防止 mixed executor 依赖长期不推进时无限消耗 token

## 10. 安全不变量清单

以下不变量必须长期成立：

- 任意工具执行都必须经过 `PermissionEngine` 与 `WorkspaceSandbox`
- 子代理工具执行不得绕过 runtime 权限链
- `todo_write` 的 schema、解析、校验必须同源一致
- Todo 状态变化必须受状态机与 revision 约束
- 依赖失败必须显式传导为 `dependency_failed`，不伪装为长期 blocked
- `ask` 决议必须有完整事件对（requested/resolved）
- session 级 allow/reject 记忆必须基于规范化 action key

## 11. 常见故障定位（按优先级）

### 11.1 Todo 长期 blocked

优先排查：

- 上游依赖是否已 `failed/canceled`（应触发 dependency_failed）
- `next_retry_at` 是否尚未到期
- executor 是否设置错误（本应 `subagent` 却是 `agent`）

### 11.2 subagent 未实际执行

优先排查：

- 是否进入 `dispatchTodos` 阶段
- `hasDispatchableSubAgentTodo` 是否为真
- `RunSubAgentTask` 是否发出 started/progress 事件
- provider/tool 调用是否被权限拒绝或 capability 拒绝

### 11.3 allow session 后仍重复 ask

优先排查：

- action key 三段是否一致（type/category/target_scope）
- 命令是否仅“格式不同但语义一致”（归一化是否命中）
- session_id 是否变化导致记忆隔离

## 12. 扩展建议（不破坏当前主链路）

- 新工具接入：先补 `permission_mapper`，再接 executor，最后补 runtime 事件测试。
- 新 Todo 状态：必须同时更新 `Valid/IsTerminal/ValidTransition` 与调度器分支。
- 新 subagent 角色：必须定义 `RolePolicy`（allowed tools / output contract）。
- 新权限策略：优先在 `PolicyEngine` 规则层完成，不在 runtime 写 if-else 特判。

## 13. 相关源码索引

- `internal/tools/todo/write.go`
- `internal/tools/todo/common.go`
- `internal/session/todo.go`
- `internal/runtime/todo_mutator.go`
- `internal/runtime/run.go`
- `internal/runtime/subagent_dispatch.go`
- `internal/runtime/subagent_run.go`
- `internal/runtime/subagent_engine.go`
- `internal/runtime/subagent_tool_executor.go`
- `internal/runtime/permission.go`
- `internal/runtime/approval/broker.go`
- `internal/subagent/scheduler.go`
- `internal/subagent/scheduler_types.go`
- `internal/tools/manager.go`
- `internal/tools/session_memory.go`
- `internal/security/policy.go`
- `internal/security/workspace.go`
