package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-llm-demo/internal/server/domain"
	servertools "go-llm-demo/internal/server/infra/tools"
)

const maxToolSteps = 8

// RunAgent 在基础聊天客户端之上执行工具循环，并输出结构化事件流。
func RunAgent(ctx context.Context, client ChatClient, messages []Message, model string) (<-chan domain.AgentEvent, error) {
	stream := make(chan domain.AgentEvent, 64)
	go func() {
		defer close(stream)
		current := append([]Message{}, messages...)
		for step := 0; step < maxToolSteps; step++ {
			emitEvent(stream, domain.AgentEvent{Type: "agent.thought", Message: fmt.Sprintf("第 %d 步：请求模型响应", step+1)})
			chunks, err := client.Chat(ctx, current, model)
			if err != nil {
				emitEvent(stream, domain.AgentEvent{Type: "stream.error", Message: err.Error()})
				return
			}

			var replyBuilder strings.Builder
			for chunk := range chunks {
				replyBuilder.WriteString(chunk)
				emitEvent(stream, domain.AgentEvent{Type: "stream.chunk", Data: map[string]interface{}{"content": chunk}})
			}

			reply := strings.TrimSpace(replyBuilder.String())
			call, ok, err := extractToolCall(reply)
			if err != nil {
				emitEvent(stream, domain.AgentEvent{Type: "tool.parse.warning", Message: fmt.Sprintf("工具调用解析失败，按普通回复处理: %v", err)})
				emitEvent(stream, domain.AgentEvent{Type: "stream.done"})
				return
			}
			if !ok {
				emitEvent(stream, domain.AgentEvent{Type: "stream.done"})
				return
			}

			emitEvent(stream, domain.AgentEvent{Type: "tool.detected", Message: fmt.Sprintf("检测到工具调用: %s", call.Tool), Data: toolCallData(call)})
			emitEvent(stream, domain.AgentEvent{Type: "tool.started", Message: formatToolProgress(call), Data: toolCallData(call)})

			result := servertools.GlobalRegistry.Execute(call)
			observation := ""
			if result.Success {
				observation = fmt.Sprintf("工具执行结果(%s): %s", result.ToolName, strings.TrimSpace(result.Output))
				emitEvent(stream, domain.AgentEvent{Type: "tool.result", Message: observation, Data: map[string]interface{}{"tool": result.ToolName, "output": result.Output, "metadata": result.Metadata}})
			} else {
				observation = fmt.Sprintf("工具执行错误(%s): %s", result.ToolName, result.Error)
				emitEvent(stream, domain.AgentEvent{Type: "tool.error", Message: observation, Data: map[string]interface{}{"tool": result.ToolName, "error": result.Error, "metadata": result.Metadata}})
			}
			current = append(current, Message{Role: "system", Content: observation}, Message{Role: "assistant", Content: ""})
		}
		emitEvent(stream, domain.AgentEvent{Type: "stream.error", Message: fmt.Sprintf("工具调用超过最大步数 %d", maxToolSteps)})
	}()
	return stream, nil
}

func emitEvent(ch chan<- domain.AgentEvent, event domain.AgentEvent) {
	ch <- event
}

func toolCallData(call domain.ToolCall) map[string]interface{} {
	return map[string]interface{}{"tool": call.Tool, "params": servertools.NormalizeParams(call.Params)}
}

func formatToolProgress(call domain.ToolCall) string {
	params := servertools.NormalizeParams(call.Params)
	if filePath, ok := params["filePath"].(string); ok && filePath != "" {
		return fmt.Sprintf("%s: 正在处理 %s", call.Tool, filePath)
	}
	if workdir, ok := params["workdir"].(string); ok && workdir != "" {
		return fmt.Sprintf("%s: 在 %s 中执行工具", call.Tool, workdir)
	}
	return fmt.Sprintf("%s: 正在执行工具...", call.Tool)
}

func extractToolCall(content string) (domain.ToolCall, bool, error) {
	candidates := []string{strings.TrimSpace(content)}
	candidates = append(candidates, extractJSONCodeBlocks(content)...)
	if obj := extractFirstJSONObject(content); obj != "" {
		candidates = append(candidates, obj)
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		var call domain.ToolCall
		if err := json.Unmarshal([]byte(candidate), &call); err != nil {
			continue
		}
		if strings.TrimSpace(call.Tool) == "" {
			continue
		}
		call.Params = servertools.NormalizeParams(call.Params)
		return call, true, nil
	}
	if strings.Contains(content, "\"tool\"") {
		return domain.ToolCall{}, false, fmt.Errorf("检测到 tool 字段但未能提取合法 JSON")
	}
	return domain.ToolCall{}, false, nil
}

func extractJSONCodeBlocks(content string) []string {
	parts := strings.Split(content, "```")
	blocks := make([]string, 0)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "json") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "json"))
		}
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			blocks = append(blocks, trimmed)
		}
	}
	return blocks
}

func extractFirstJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return ""
}
