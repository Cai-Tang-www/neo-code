package runtime

import (
	"context"
	"strings"

	providertypes "neo-code/internal/provider/types"
	"neo-code/internal/runtime/acceptance"
	"neo-code/internal/runtime/controlplane"
	"neo-code/internal/runtime/verify"
	agentsession "neo-code/internal/session"
)

const finalContinueReminder = "There are unfinished required todos or unmet acceptance checks. Continue execution. Do not finalize yet."

var taskTypeKeywordRules = []struct {
	taskType string
	keywords []string
}{
	{
		taskType: "fix_bug",
		keywords: []string{"fix bug", "bugfix", "修复", "故障", "问题"},
	},
	{
		taskType: "refactor",
		keywords: []string{"refactor", "重构"},
	},
	{
		taskType: "edit_code",
		keywords: []string{"edit code", "modify code", "patch", "修改代码", "改代码", "打补丁"},
	},
	{
		taskType: "create_file",
		keywords: []string{"create file", "scaffold", "创建文件", "新增文件", "脚手架"},
	},
	{
		taskType: "docs",
		keywords: []string{"docs", "documentation", "文档", "说明"},
	},
	{
		taskType: "config",
		keywords: []string{"config", "yaml", "json", "配置"},
	},
}

// beforeAcceptFinal 在 runtime 接受模型 final 前执行双门控验收。
func (s *Service) beforeAcceptFinal(
	ctx context.Context,
	state *runState,
	snapshot TurnBudgetSnapshot,
	assistant providertypes.Message,
	completionPassed bool,
) (acceptance.AcceptanceDecision, error) {
	if state == nil {
		return acceptance.AcceptanceDecision{}, nil
	}

	verificationCfg := snapshot.Config.Runtime.Verification.Clone()
	if !verificationCfg.FinalInterceptValue() {
		return acceptance.AcceptanceDecision{
			Status:             acceptance.AcceptanceAccepted,
			StopReason:         controlplane.StopReasonCompatibilityFallback,
			UserVisibleSummary: "已通过兼容路径接受 final（final_intercept 关闭）。",
			InternalSummary:    "verification final intercept disabled, compatibility fallback accepted",
			HasProgress:        true,
		}, nil
	}

	policy := acceptance.DefaultPolicy{
		Executor: verify.PolicyCommandExecutor{},
	}
	engine := acceptance.NewEngine(policy)

	maxNoProgress := verificationCfg.MaxNoProgress
	if maxNoProgress <= 0 {
		maxNoProgress = 3
	}
	input := acceptance.FinalAcceptanceInput{
		CompletionGate: acceptance.CompletionGateDecision{
			Passed: completionPassed,
			Reason: string(state.completion.CompletionBlockedReason),
		},
		VerificationInput: verify.FinalVerifyInput{
			SessionID:          state.session.ID,
			RunID:              state.runID,
			TaskID:             state.taskID,
			Workdir:            snapshot.Workdir,
			Messages:           buildVerifyMessages(state.session.Messages),
			Todos:              buildVerifyTodos(state.session.Todos),
			LastAssistantFinal: renderPartsForVerification(assistant.Parts),
			ToolResults:        nil,
			RuntimeState: verify.RuntimeStateSnapshot{
				Turn:                 state.turn,
				MaxTurns:             resolveRuntimeMaxTurns(snapshot.Config.Runtime),
				MaxTurnsReached:      state.maxTurnsReached,
				FinalInterceptStreak: state.finalInterceptStreak,
			},
			Metadata: map[string]any{
				"task_type": inferTaskType(state),
			},
			VerificationConfig: verificationCfg,
		},
		NoProgressExceeded: state.finalInterceptStreak >= maxNoProgress,
		MaxTurnsReached:    state.maxTurnsReached,
		MaxTurnsLimit:      state.maxTurnsLimit,
	}

	return engine.EvaluateFinal(ctx, input)
}

// recordAcceptanceTerminal 将 acceptance 输出映射为 runtime 唯一终态记录。
func recordAcceptanceTerminal(state *runState, decision acceptance.AcceptanceDecision) {
	if state == nil {
		return
	}
	status := acceptance.TerminalStatusFromAcceptance(decision.Status)
	state.markTerminalDecision(status, decision.StopReason, strings.TrimSpace(decision.InternalSummary))
}

// buildVerifyTodos 将 session todo 转换为 verifier 快照。
func buildVerifyTodos(items []agentsession.TodoItem) []verify.TodoSnapshot {
	if len(items) == 0 {
		return nil
	}
	todos := make([]verify.TodoSnapshot, 0, len(items))
	for _, item := range items {
		todos = append(todos, verify.TodoSnapshot{
			ID:            trimVerifyText(item.ID),
			Content:       trimVerifyText(item.Content),
			Status:        trimVerifyText(string(item.Status)),
			Required:      item.RequiredValue(),
			BlockedReason: string(item.BlockedReasonValue()),
			RetryCount:    item.RetryCount,
			RetryLimit:    item.RetryLimit,
			FailureReason: trimVerifyText(item.FailureReason),
		})
	}
	return todos
}

// buildVerifyMessages 将会话消息压缩为 verifier 所需最小快照。
func buildVerifyMessages(messages []providertypes.Message) []verify.MessageLike {
	if len(messages) == 0 {
		return nil
	}
	out := make([]verify.MessageLike, 0, len(messages))
	for _, message := range messages {
		out = append(out, verify.MessageLike{
			Role:    trimVerifyText(message.Role),
			Content: renderPartsForVerification(message.Parts),
		})
	}
	return out
}

// renderPartsForVerification 将消息分片合并为 verifier 侧可读文本。
func renderPartsForVerification(parts []providertypes.ContentPart) string {
	if len(parts) == 0 {
		return ""
	}
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Kind != providertypes.ContentPartText {
			continue
		}
		text := trimVerifyText(part.Text)
		if text == "" {
			continue
		}
		segments = append(segments, text)
	}
	return strings.Join(segments, "\n")
}

// inferTaskType 基于 task_id 与 task_state 文本推断当前任务类型。
func inferTaskType(state *runState) string {
	if state == nil {
		return "unknown"
	}
	corpus := strings.ToLower(strings.TrimSpace(
		state.taskID + " " + state.session.TaskState.Goal + " " + state.session.TaskState.NextStep,
	))
	for _, rule := range taskTypeKeywordRules {
		if containsAnyKeyword(corpus, rule.keywords...) {
			return rule.taskType
		}
	}
	return "unknown"
}

// applyAcceptanceResultProgress 根据 acceptance 输出更新 final 拦截熔断计数器。
func applyAcceptanceResultProgress(state *runState, decision acceptance.AcceptanceDecision) {
	if state == nil {
		return
	}
	switch decision.Status {
	case acceptance.AcceptanceContinue:
		if decision.HasProgress {
			state.finalInterceptStreak = 0
			return
		}
		state.finalInterceptStreak++
	default:
		state.finalInterceptStreak = 0
	}
}

// trimVerifyText 统一裁剪 verifier 快照里的字符串字段。
func trimVerifyText(value string) string {
	return strings.TrimSpace(value)
}

// containsAnyKeyword 判断语料中是否命中任一关键词。
func containsAnyKeyword(corpus string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(corpus, keyword) {
			return true
		}
	}
	return false
}
