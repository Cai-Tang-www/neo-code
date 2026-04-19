package spawnsubagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	agentsession "neo-code/internal/session"
	"neo-code/internal/tools"
)

const (
	maxSpawnArgumentsBytes = 64 * 1024
	maxSpawnItems          = 64
	maxSpawnTextLen        = 1024
	maxSpawnListItems      = 64
)

type spawnInput struct {
	Items []spawnItem `json:"items"`
}

type spawnItem struct {
	ID           string   `json:"id"`
	Content      string   `json:"content"`
	Dependencies []string `json:"dependencies,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	Acceptance   []string `json:"acceptance,omitempty"`
	RetryLimit   int      `json:"retry_limit,omitempty"`
}

// Tool 定义 spawn_subagent 工具：仅创建 executor=subagent 的 Todo 项，不直接执行子代理。
type Tool struct{}

// New 返回 spawn_subagent 工具实例。
func New() *Tool {
	return &Tool{}
}

// Name 返回工具唯一名称。
func (t *Tool) Name() string {
	return tools.ToolNameSpawnSubAgent
}

// Description 返回工具描述。
func (t *Tool) Description() string {
	return "Create subagent todos only (executor=subagent) and let runtime scheduler execute them later."
}

// Schema 返回 spawn_subagent 的参数定义，只暴露“创建 Todo”能力。
func (t *Tool) Schema() map[string]any {
	itemSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{
				"type": "string",
			},
			"content": map[string]any{
				"type": "string",
			},
			"dependencies": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			"priority": map[string]any{
				"type": "integer",
			},
			"acceptance": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
			"retry_limit": map[string]any{
				"type": "integer",
			},
		},
		"required": []string{"id", "content"},
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type":  "array",
				"items": itemSchema,
			},
		},
		"required": []string{"items"},
	}
}

// MicroCompactPolicy 声明 spawn_subagent 结果默认参与 micro compact。
func (t *Tool) MicroCompactPolicy() tools.MicroCompactPolicy {
	return tools.MicroCompactPolicyCompact
}

// Execute 解析并校验入参后，按依赖顺序创建 subagent Todo。
func (t *Tool) Execute(ctx context.Context, call tools.ToolCallInput) (tools.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), "", nil), err
	}
	if call.SessionMutator == nil {
		err := errors.New("spawn_subagent: session mutator is unavailable")
		result := tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), "", nil)
		result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
		return result, err
	}

	input, err := parseSpawnInput(call.Arguments)
	if err != nil {
		result := tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), err.Error(), nil)
		result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
		return result, err
	}

	ordered, err := resolveSpawnOrder(call.SessionMutator.ListTodos(), input.Items)
	if err != nil {
		result := tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), err.Error(), nil)
		result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
		return result, err
	}

	created := make([]string, 0, len(ordered))
	for _, item := range ordered {
		todo := agentsession.TodoItem{
			ID:           item.ID,
			Content:      item.Content,
			Status:       agentsession.TodoStatusPending,
			Dependencies: append([]string(nil), item.Dependencies...),
			Priority:     item.Priority,
			Executor:     agentsession.TodoExecutorSubAgent,
			Acceptance:   append([]string(nil), item.Acceptance...),
			RetryLimit:   item.RetryLimit,
		}
		if err := call.SessionMutator.AddTodo(todo); err != nil {
			result := tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), err.Error(), nil)
			result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
			return result, err
		}
		created = append(created, item.ID)
	}

	result := tools.ToolResult{
		Name:    t.Name(),
		Content: renderSpawnResult(created),
		Metadata: map[string]any{
			"created_count": len(created),
			"created_ids":   created,
		},
	}
	result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
	return result, nil
}

// parseSpawnInput 负责解析并校验 spawn_subagent 输入。
func parseSpawnInput(raw []byte) (spawnInput, error) {
	if len(raw) == 0 {
		return spawnInput{}, errors.New("spawn_subagent: arguments is empty")
	}
	if len(raw) > maxSpawnArgumentsBytes {
		return spawnInput{}, fmt.Errorf(
			"spawn_subagent: arguments payload exceeds %d bytes",
			maxSpawnArgumentsBytes,
		)
	}

	var input spawnInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return spawnInput{}, fmt.Errorf("spawn_subagent: parse arguments: %w", err)
	}
	if len(input.Items) == 0 {
		return spawnInput{}, errors.New("spawn_subagent: items is empty")
	}
	if len(input.Items) > maxSpawnItems {
		return spawnInput{}, fmt.Errorf("spawn_subagent: items exceeds max length %d", maxSpawnItems)
	}

	for idx := range input.Items {
		item := &input.Items[idx]
		item.ID = strings.TrimSpace(item.ID)
		item.Content = strings.TrimSpace(item.Content)
		item.Dependencies = normalizeStringList(item.Dependencies)
		item.Acceptance = normalizeStringList(item.Acceptance)
		if item.ID == "" {
			return spawnInput{}, fmt.Errorf("spawn_subagent: items[%d].id is empty", idx)
		}
		if item.Content == "" {
			return spawnInput{}, fmt.Errorf("spawn_subagent: items[%d].content is empty", idx)
		}
		if len(item.ID) > maxSpawnTextLen {
			return spawnInput{}, fmt.Errorf("spawn_subagent: items[%d].id exceeds max length %d", idx, maxSpawnTextLen)
		}
		if len(item.Content) > maxSpawnTextLen {
			return spawnInput{}, fmt.Errorf("spawn_subagent: items[%d].content exceeds max length %d", idx, maxSpawnTextLen)
		}
		if len(item.Dependencies) > maxSpawnListItems {
			return spawnInput{}, fmt.Errorf(
				"spawn_subagent: items[%d].dependencies exceeds max items %d",
				idx,
				maxSpawnListItems,
			)
		}
		if len(item.Acceptance) > maxSpawnListItems {
			return spawnInput{}, fmt.Errorf(
				"spawn_subagent: items[%d].acceptance exceeds max items %d",
				idx,
				maxSpawnListItems,
			)
		}
		for depIdx := range item.Dependencies {
			if len(item.Dependencies[depIdx]) > maxSpawnTextLen {
				return spawnInput{}, fmt.Errorf(
					"spawn_subagent: items[%d].dependencies[%d] exceeds max length %d",
					idx,
					depIdx,
					maxSpawnTextLen,
				)
			}
		}
		for accIdx := range item.Acceptance {
			if len(item.Acceptance[accIdx]) > maxSpawnTextLen {
				return spawnInput{}, fmt.Errorf(
					"spawn_subagent: items[%d].acceptance[%d] exceeds max length %d",
					idx,
					accIdx,
					maxSpawnTextLen,
				)
			}
		}
		if item.RetryLimit < 0 {
			return spawnInput{}, fmt.Errorf("spawn_subagent: items[%d].retry_limit must be >= 0", idx)
		}
	}
	return input, nil
}

// resolveSpawnOrder 在校验依赖可达后，返回可安全写入会话的拓扑有序任务列表。
func resolveSpawnOrder(existing []agentsession.TodoItem, items []spawnItem) ([]spawnItem, error) {
	existingSet := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		existingSet[item.ID] = struct{}{}
	}

	itemsByID := make(map[string]spawnItem, len(items))
	inDegree := make(map[string]int, len(items))
	dependents := make(map[string][]string, len(items))
	for _, item := range items {
		if _, exists := existingSet[item.ID]; exists {
			return nil, fmt.Errorf("spawn_subagent: todo %q already exists", item.ID)
		}
		if _, exists := itemsByID[item.ID]; exists {
			return nil, fmt.Errorf("spawn_subagent: duplicate todo id %q", item.ID)
		}
		itemsByID[item.ID] = item
		inDegree[item.ID] = 0
	}

	for _, item := range items {
		for _, depID := range item.Dependencies {
			if depID == item.ID {
				return nil, fmt.Errorf("spawn_subagent: todo %q cannot depend on itself", item.ID)
			}
			if _, exists := existingSet[depID]; exists {
				continue
			}
			if _, exists := itemsByID[depID]; !exists {
				return nil, fmt.Errorf("spawn_subagent: todo %q references unknown dependency %q", item.ID, depID)
			}
			inDegree[item.ID]++
			dependents[depID] = append(dependents[depID], item.ID)
		}
	}

	ready := make([]string, 0, len(items))
	for id, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)

	ordered := make([]spawnItem, 0, len(items))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		ordered = append(ordered, itemsByID[id])

		next := dependents[id]
		sort.Strings(next)
		for _, depID := range next {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				ready = append(ready, depID)
			}
		}
		sort.Strings(ready)
	}

	if len(ordered) != len(items) {
		return nil, errors.New("spawn_subagent: cyclic dependencies detected")
	}
	return ordered, nil
}

// normalizeStringList 统一清理字符串列表并去重，保持输入顺序稳定。
func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// renderSpawnResult 输出创建结果摘要，便于模型下一轮读取已新增任务 ID。
func renderSpawnResult(created []string) string {
	lines := []string{
		"spawn_subagent result",
		fmt.Sprintf("created_count: %d", len(created)),
	}
	if len(created) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "created_ids:")
	for _, id := range created {
		lines = append(lines, "- "+id)
	}
	return strings.Join(lines, "\n")
}
