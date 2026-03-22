package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-llm-demo/internal/server/domain"
	"go-llm-demo/internal/server/infra/tools"
)

type fakeMemoryService struct{}

func (fakeMemoryService) BuildContext(ctx context.Context, userInput string) (string, error) {
	return "", nil
}
func (fakeMemoryService) Save(ctx context.Context, userInput, reply string) error { return nil }
func (fakeMemoryService) GetStats(ctx context.Context) (*domain.MemoryStats, error) {
	return &domain.MemoryStats{}, nil
}
func (fakeMemoryService) Clear(ctx context.Context) error        { return nil }
func (fakeMemoryService) ClearSession(ctx context.Context) error { return nil }

type fakeWorkingMemoryService struct{}

func (fakeWorkingMemoryService) BuildContext(ctx context.Context, messages []domain.Message) (string, error) {
	return "", nil
}
func (fakeWorkingMemoryService) Refresh(ctx context.Context, messages []domain.Message) error {
	return nil
}
func (fakeWorkingMemoryService) Clear(ctx context.Context) error { return nil }

type fakeRoleService struct{}

func (fakeRoleService) GetActivePrompt(ctx context.Context) (string, error) { return "", nil }
func (fakeRoleService) SetActive(ctx context.Context, roleID string) error  { return nil }
func (fakeRoleService) List(ctx context.Context) ([]domain.Role, error)     { return nil, nil }
func (fakeRoleService) Create(ctx context.Context, name, desc, prompt string) (*domain.Role, error) {
	return nil, nil
}
func (fakeRoleService) Delete(ctx context.Context, id string) error { return nil }

type fakeProvider struct {
	replies [][]string
	errs    []error
	calls   [][]domain.Message
}

func (f *fakeProvider) GetModelName() string { return "fake" }
func (f *fakeProvider) Chat(ctx context.Context, messages []domain.Message) (<-chan string, error) {
	idx := len(f.calls)
	f.calls = append(f.calls, append([]domain.Message{}, messages...))
	if idx < len(f.errs) && f.errs[idx] != nil {
		return nil, f.errs[idx]
	}
	out := make(chan string, len(f.replies[idx]))
	for _, chunk := range f.replies[idx] {
		out <- chunk
	}
	close(out)
	return out, nil
}

func TestExtractToolCallSupportsJSONFenceAndSnakeCase(t *testing.T) {
	reply := "```json\n{\"tool\":\"read\",\"params\":{\"file_path\":\"README.md\",\"offset\":1}}\n```"
	call, ok, err := extractToolCall(reply)
	if err != nil {
		t.Fatalf("extractToolCall returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected tool call to be extracted")
	}
	if call.Tool != "read" {
		t.Fatalf("expected tool read, got %s", call.Tool)
	}
	if _, exists := call.Params["filePath"]; !exists {
		t.Fatalf("expected snake_case parameter to be normalized: %#v", call.Params)
	}
}

func TestExtractToolCallIgnoresNaturalLanguageWithBraces(t *testing.T) {
	reply := "你可以在 config {path} 里继续配置，但这里不需要调用工具。"
	_, ok, err := extractToolCall(reply)
	if err != nil {
		t.Fatalf("expected non-JSON natural language to be ignored, got %v", err)
	}
	if ok {
		t.Fatalf("expected natural language with braces not to be treated as a tool call")
	}
}

func TestExecuteToolWithRetryValidatesSchema(t *testing.T) {
	tools.Initialize()
	svc := &chatServiceImpl{}
	_, err := svc.executeToolWithRetry(&toolCallPayload{Tool: "read", Params: map[string]interface{}{"unknown": "x"}})
	if err == nil || !strings.Contains(err.Error(), "未知参数") {
		t.Fatalf("expected schema validation error, got %v", err)
	}
}

func TestCollectAssistantReplyWithRetryRetriesProviderFailures(t *testing.T) {
	provider := &fakeProvider{
		errs:    []error{errors.New("temporary failure"), nil},
		replies: [][]string{nil, {"修复后的响应"}},
	}
	svc := &chatServiceImpl{provider: provider}
	reply, err := svc.collectAssistantReplyWithRetry(context.Background(), []domain.Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("collectAssistantReplyWithRetry returned error: %v", err)
	}
	if reply != "修复后的响应" {
		t.Fatalf("expected retried response, got %q", reply)
	}
	if len(provider.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(provider.calls))
	}
}

func TestSendRunsServiceLevelReActLoop(t *testing.T) {
	provider := &fakeProvider{replies: [][]string{{`{"tool":"list","params":{"path":"."}}`}, {"已完成目录检查"}}}
	svc := NewChatService(fakeMemoryService{}, fakeWorkingMemoryService{}, fakeRoleService{}, provider)
	stream, err := svc.Send(context.Background(), &domain.ChatRequest{Messages: []domain.Message{{Role: "user", Content: "看看当前目录"}}})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	var got strings.Builder
	for chunk := range stream {
		got.WriteString(chunk)
	}
	if got.String() != "已完成目录检查" {
		t.Fatalf("expected final answer, got %q", got.String())
	}
	if len(provider.calls) != 2 {
		t.Fatalf("expected 2 provider calls, got %d", len(provider.calls))
	}
	secondCall := provider.calls[1]
	joined := make([]string, 0, len(secondCall))
	for _, msg := range secondCall {
		joined = append(joined, msg.Content)
	}
	joinedText := strings.Join(joined, "\n")
	if !strings.Contains(joinedText, "工具执行结果") {
		t.Fatalf("expected tool result to be fed back into second model call, got %s", joinedText)
	}
	if !strings.Contains(joinedText, `{"tool":"list"`) {
		t.Fatalf("expected original tool JSON to be preserved in assistant history, got %s", joinedText)
	}
}
