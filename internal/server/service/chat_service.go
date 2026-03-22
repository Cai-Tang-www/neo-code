package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-llm-demo/internal/server/domain"
	"go-llm-demo/internal/server/infra/tools"
)

const (
	maxToolSteps          = 6
	maxToolExecutionRetry = 2
	maxAIResponseRetry    = 2
)

type chatServiceImpl struct {
	memorySvc  domain.MemoryService
	workingSvc domain.WorkingMemoryService
	roleSvc    domain.RoleService
	provider   domain.ChatProvider
}

type toolCallPayload struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

func NewChatService(memorySvc domain.MemoryService, workingSvc domain.WorkingMemoryService, roleSvc domain.RoleService, provider domain.ChatProvider) domain.ChatGateway {
	tools.Initialize()
	return &chatServiceImpl{
		memorySvc:  memorySvc,
		workingSvc: workingSvc,
		roleSvc:    roleSvc,
		provider:   provider,
	}
}

func (s *chatServiceImpl) Send(ctx context.Context, req *domain.ChatRequest) (<-chan string, error) {
	preparedMessages, userInput, err := s.prepareMessages(ctx, req.Messages)
	if err != nil {
		return nil, err
	}

	resultChan := make(chan string)
	go func() {
		defer close(resultChan)

		messages := append([]domain.Message{}, preparedMessages...)
		finalReply := ""

		for step := 0; step < maxToolSteps; step++ {
			reply, chatErr := s.collectAssistantReplyWithRetry(ctx, messages)
			if chatErr != nil {
				finalReply = fmt.Sprintf("抱歉，AI 响应失败：%v", chatErr)
				break
			}

			call, ok, extractErr := extractToolCall(reply)
			if extractErr != nil {
				messages = append(messages,
					domain.Message{Role: "assistant", Content: reply},
					domain.Message{Role: "system", Content: fmt.Sprintf("工具调用协议解析失败：%v。请严格按照 JSON 协议重试，或直接输出最终答案。", extractErr)},
				)
				continue
			}
			if !ok {
				finalReply = reply
				break
			}

			messages = append(messages, domain.Message{Role: "assistant", Content: reply})
			toolResult, execErr := s.executeToolWithRetry(call)
			if execErr != nil {
				messages = append(messages, domain.Message{Role: "system", Content: fmt.Sprintf("工具执行失败：%v。请修正参数、尝试其他工具，或直接给出不调用工具的答复。", execErr)})
				continue
			}

			toolJSON, marshalErr := json.Marshal(toolResult)
			if marshalErr != nil {
				messages = append(messages, domain.Message{Role: "system", Content: fmt.Sprintf("工具执行完成，但结果编码失败：%v。请总结当前结果。", marshalErr)})
				continue
			}
			messages = append(messages, domain.Message{Role: "system", Content: fmt.Sprintf("工具执行结果：%s", string(toolJSON))})
		}

		if strings.TrimSpace(finalReply) == "" {
			finalReply = "抱歉，我未能在限定步数内完成 ReAct 流程，请调整请求后重试。"
		}

		s.streamString(resultChan, finalReply)
		s.persistMemory(req.Messages, userInput, finalReply)
	}()

	return resultChan, nil
}

func (s *chatServiceImpl) prepareMessages(ctx context.Context, incoming []domain.Message) ([]domain.Message, string, error) {
	messages := append([]domain.Message{}, incoming...)

	rolePrompt, err := s.roleSvc.GetActivePrompt(ctx)
	if err != nil {
		fmt.Printf("获取角色提示失败：%v\n", err)
	}
	if rolePrompt != "" && !hasSystemMessage(messages) {
		messages = append([]domain.Message{{Role: "system", Content: rolePrompt}}, messages...)
	}

	userInput := s.latestUserInput(messages)
	workingContext := ""
	if s.workingSvc != nil {
		workingContext, err = s.workingSvc.BuildContext(ctx, messages)
		if err != nil {
			return nil, "", err
		}
	}

	memoryContext := ""
	if userInput != "" {
		memoryContext, err = s.memorySvc.BuildContext(ctx, userInput)
		if err != nil {
			return nil, "", err
		}
	}

	toolProtocol := buildToolProtocolPrompt()
	combinedContext := joinContextBlocks(workingContext, memoryContext, toolProtocol)
	if combinedContext != "" {
		if len(messages) > 0 && messages[0].Role == "system" {
			messages[0].Content = joinContextBlocks(messages[0].Content, combinedContext)
		} else {
			messages = append([]domain.Message{{Role: "system", Content: combinedContext}}, messages...)
		}
	}

	return messages, userInput, nil
}

func buildToolProtocolPrompt() string {
	schemas := tools.GlobalRegistry.Schemas()
	return strings.TrimSpace(`你可以在需要时调用工具。
当且仅当你确定需要调用工具时，必须只输出一个 JSON 对象，不要输出 Markdown、解释文本或额外字符。
JSON 格式如下：
{"tool":"工具名","params":{...}}
如果不需要调用工具，请直接输出对用户可见的最终答复。
调用工具前请确保参数名与 schema 完全一致，并且不要传递未声明参数。可用工具如下：
` + tools.FormatSchemasForPrompt(schemas))
}

func hasSystemMessage(messages []domain.Message) bool {
	for _, msg := range messages {
		if msg.Role == "system" {
			return true
		}
	}
	return false
}

func (s *chatServiceImpl) collectAssistantReplyWithRetry(ctx context.Context, messages []domain.Message) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAIResponseRetry; attempt++ {
		reply, err := s.collectAssistantReply(ctx, messages)
		if err == nil && strings.TrimSpace(reply) != "" {
			return reply, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("AI 没有返回有效内容")
		}
	}
	return "", lastErr
}

func (s *chatServiceImpl) collectAssistantReply(ctx context.Context, messages []domain.Message) (string, error) {
	out, err := s.provider.Chat(ctx, messages)
	if err != nil {
		return "", err
	}
	var replyBuilder strings.Builder
	for chunk := range out {
		replyBuilder.WriteString(chunk)
	}
	return replyBuilder.String(), nil
}

func extractToolCall(reply string) (*toolCallPayload, bool, error) {
	candidate, ok := extractJSONObjectCandidate(reply)
	if !ok {
		return nil, false, nil
	}

	var payload toolCallPayload
	if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
		return nil, false, fmt.Errorf("JSON 解析失败: %w", err)
	}
	if strings.TrimSpace(payload.Tool) == "" {
		return nil, false, fmt.Errorf("缺少 tool 字段")
	}
	if payload.Params == nil {
		payload.Params = map[string]interface{}{}
	}
	payload.Params = normalizeParamKeys(payload.Params)
	return &payload, true, nil
}

func extractJSONObjectCandidate(reply string) (string, bool) {
	trimmed := strings.TrimSpace(reply)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(trimmed, "```") {
		parts := strings.SplitN(trimmed, "\n", 2)
		if len(parts) < 2 {
			return "", false
		}
		body := parts[1]
		end := strings.LastIndex(body, "```")
		if end < 0 {
			return "", false
		}
		trimmed = strings.TrimSpace(body[:end])
	}

	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return "", false
	}
	return trimmed, true
}

func normalizeParamKeys(params map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(params))
	for key, value := range params {
		parts := strings.Split(key, "_")
		camelKey := parts[0]
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				camelKey += strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		result[camelKey] = value
	}
	return result
}

func (s *chatServiceImpl) executeToolWithRetry(call *toolCallPayload) (*tools.ToolResult, error) {
	tool := tools.GlobalRegistry.Get(call.Tool)
	if tool == nil {
		return nil, fmt.Errorf("不支持的工具: %s", call.Tool)
	}
	if err := tools.ValidateParams(tool.Schema(), call.Params); err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= maxToolExecutionRetry; attempt++ {
		result := tool.Run(call.Params)
		if result != nil && result.Success {
			if result.Metadata == nil {
				result.Metadata = map[string]interface{}{}
			}
			result.Metadata["attempt"] = attempt
			return result, nil
		}
		if result != nil && result.Error != "" {
			lastErr = fmt.Errorf(result.Error)
		} else {
			lastErr = fmt.Errorf("工具执行失败")
		}
	}
	return nil, fmt.Errorf("工具 %s 在 %d 次尝试后仍失败: %w", call.Tool, maxToolExecutionRetry, lastErr)
}

func (s *chatServiceImpl) streamString(out chan<- string, text string) {
	if text == "" {
		return
	}
	for _, r := range text {
		out <- string(r)
	}
}

func (s *chatServiceImpl) persistMemory(originalMessages []domain.Message, userInput, finalReply string) {
	if userInput == "" || strings.TrimSpace(finalReply) == "" {
		return
	}
	if s.workingSvc != nil {
		updatedMessages := append([]domain.Message{}, originalMessages...)
		updatedMessages = append(updatedMessages, domain.Message{Role: "assistant", Content: finalReply})
		if err := s.workingSvc.Refresh(context.Background(), updatedMessages); err != nil {
			fmt.Printf("工作记忆刷新失败：%v\n", err)
		}
	}
	if err := s.memorySvc.Save(context.Background(), userInput, finalReply); err != nil {
		fmt.Printf("记忆保存失败：%v\n", err)
	}
}

func (s *chatServiceImpl) latestUserInput(messages []domain.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func joinContextBlocks(blocks ...string) string {
	filtered := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		filtered = append(filtered, block)
	}
	return strings.Join(filtered, "\n\n")
}
