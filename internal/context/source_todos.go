package context

import (
	"context"
	"fmt"
	"sort"
	"strings"

	agentsession "neo-code/internal/session"
)

const (
	maxPromptTodos         = 24
	maxPromptTerminalTodos = 8
	maxPromptTodoIDLength  = 80
	maxPromptTodoTextLen   = 240
	maxPromptTodoDeps      = 8
	maxPromptExecutorLen   = 32
	maxPromptOwnerLen      = 64
)

// todosSource 负责把会话 Todo 摘要渲染为 prompt section。
type todosSource struct{}

// Sections 渲染非终态 Todo，按状态与优先级排序后注入上下文。
func (todosSource) Sections(ctx context.Context, input BuildInput) ([]promptSection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(input.Todos) == 0 {
		return nil, nil
	}

	active := make([]agentsession.TodoItem, 0, len(input.Todos))
	terminal := make([]agentsession.TodoItem, 0, len(input.Todos))
	for _, item := range input.Todos {
		if !item.Status.IsTerminal() {
			active = append(active, item.Clone())
			continue
		}
		terminal = append(terminal, item.Clone())
	}
	if len(active) == 0 && len(terminal) == 0 {
		return nil, nil
	}

	lines := make([]string, 0, len(active)+len(terminal)+8)
	if len(active) > 0 {
		sort.SliceStable(active, func(i, j int) bool {
			left := todoStatusRank(active[i].Status)
			right := todoStatusRank(active[j].Status)
			if left != right {
				return left < right
			}
			if active[i].Priority != active[j].Priority {
				return active[i].Priority > active[j].Priority
			}
			return active[i].CreatedAt.Before(active[j].CreatedAt)
		})
		if len(active) > maxPromptTodos {
			active = active[:maxPromptTodos]
		}
		for _, item := range active {
			lines = append(lines, renderPromptTodoLines(item)...)
		}
	}

	if len(terminal) > 0 {
		sort.SliceStable(terminal, func(i, j int) bool {
			if !terminal[i].UpdatedAt.Equal(terminal[j].UpdatedAt) {
				return terminal[i].UpdatedAt.After(terminal[j].UpdatedAt)
			}
			if terminal[i].Priority != terminal[j].Priority {
				return terminal[i].Priority > terminal[j].Priority
			}
			return terminal[i].ID < terminal[j].ID
		})
		if len(terminal) > maxPromptTerminalTodos {
			terminal = terminal[:maxPromptTerminalTodos]
		}
		lines = append(lines, "terminal_summary:")
		for _, item := range terminal {
			lines = append(lines, renderPromptTodoTerminalLine(item))
		}
	}

	return []promptSection{
		{
			Title:   "Todo State",
			Content: strings.Join(lines, "\n"),
		},
	}, nil
}

// renderPromptTodoLines 渲染单个非终态 Todo 的主行与附加字段，确保上下文摘要结构一致。
func renderPromptTodoLines(item agentsession.TodoItem) []string {
	id := sanitizePromptValue(item.ID, maxPromptTodoIDLength)
	content := sanitizePromptValue(item.Content, maxPromptTodoTextLen)
	lines := []string{
		fmt.Sprintf("- [%s] id=%q (p=%d, rev=%d) content=%q", item.Status, id, item.Priority, item.Revision, content),
	}
	if len(item.Dependencies) > 0 {
		deps := sanitizePromptList(item.Dependencies, maxPromptTodoDeps, maxPromptTodoIDLength)
		quotedDeps := make([]string, 0, len(deps))
		for _, dep := range deps {
			quotedDeps = append(quotedDeps, fmt.Sprintf("%q", dep))
		}
		lines = append(lines, fmt.Sprintf("  deps: %s", strings.Join(quotedDeps, ", ")))
	}
	executor := sanitizePromptValue(item.Executor, maxPromptExecutorLen)
	if executor != "" {
		lines = append(lines, fmt.Sprintf("  executor: %q", executor))
	}
	if strings.TrimSpace(item.OwnerType) != "" || strings.TrimSpace(item.OwnerID) != "" {
		ownerType := sanitizePromptValue(item.OwnerType, maxPromptOwnerLen)
		ownerID := sanitizePromptValue(item.OwnerID, maxPromptOwnerLen)
		lines = append(lines, fmt.Sprintf("  owner: type=%q id=%q", ownerType, ownerID))
	}
	return lines
}

// renderPromptTodoTerminalLine 渲染终态摘要行，帮助模型在下一轮感知已完成/失败任务的增量事实。
func renderPromptTodoTerminalLine(item agentsession.TodoItem) string {
	id := sanitizePromptValue(item.ID, maxPromptTodoIDLength)
	content := sanitizePromptValue(item.Content, maxPromptTodoTextLen)
	reason := sanitizePromptValue(item.FailureReason, maxPromptTodoTextLen)
	line := fmt.Sprintf("- [%s] id=%q content=%q", item.Status, id, content)
	if reason != "" {
		line += fmt.Sprintf(" reason=%q", reason)
	}
	if len(item.Artifacts) > 0 {
		artifacts := sanitizePromptList(item.Artifacts, 3, maxPromptTodoTextLen)
		if len(artifacts) > 0 {
			line += fmt.Sprintf(" artifacts=%q", strings.Join(artifacts, ", "))
		}
	}
	return line
}

// sanitizePromptValue 对 Todo 文本字段做去控制字符、折叠空白与长度截断，降低提示注入风险。
func sanitizePromptValue(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxLen <= 0 {
		return ""
	}
	parts := strings.Fields(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return ' '
		}
		return r
	}, value))
	normalized := strings.Join(parts, " ")
	runes := []rune(normalized)
	if len(runes) <= maxLen {
		return normalized
	}
	return string(runes[:maxLen])
}

// sanitizePromptList 对文本列表做逐项规范化、去重和数量限制，避免单条 Todo 扩大注入面。
func sanitizePromptList(values []string, maxItems int, itemMaxLen int) []string {
	if len(values) == 0 || maxItems <= 0 || itemMaxLen <= 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := sanitizePromptValue(raw, itemMaxLen)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) >= maxItems {
			break
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// todoStatusRank 计算 Todo 状态排序优先级，值越小优先级越高。
func todoStatusRank(status agentsession.TodoStatus) int {
	switch status {
	case agentsession.TodoStatusInProgress:
		return 0
	case agentsession.TodoStatusBlocked:
		return 1
	case agentsession.TodoStatusPending:
		return 2
	default:
		return 3
	}
}
