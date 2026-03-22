package core

import (
	"context"
	"errors"
	"testing"

	"go-llm-demo/internal/tui/infra"
)

type fakeChatClient struct{}

func (fakeChatClient) Chat(context.Context, []infra.Message, string) (<-chan string, error) {
	return nil, errors.New("not implemented")
}

func (fakeChatClient) GetMemoryStats(context.Context) (*infra.MemoryStats, error) {
	return &infra.MemoryStats{}, nil
}

func (fakeChatClient) ClearMemory(context.Context) error {
	return nil
}

func (fakeChatClient) ClearSessionMemory(context.Context) error {
	return nil
}

func (fakeChatClient) ListModels() []string {
	return nil
}

func (fakeChatClient) DefaultModel() string {
	return "test-model"
}

func TestBuildMessagesSkipsEmptyAssistantPlaceholder(t *testing.T) {
	m := Model{
		messages: []Message{
			{Role: "system", Content: "persona"},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: ""},
		},
	}

	got := m.buildMessages()
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Role != "system" || got[1].Role != "user" {
		t.Fatalf("unexpected message order: %+v", got)
	}
	if got[1].Content != "hello" {
		t.Fatalf("expected user message to be preserved, got %+v", got[1])
	}
}

func TestStreamErrorReplacesTrailingPlaceholder(t *testing.T) {
	m := Model{
		historyTurns: 6,
		messages: []Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: ""},
		},
	}

	updated, _ := m.Update(StreamErrorMsg{Err: errors.New("boom")})
	got := updated.(Model)
	if len(got.messages) != 2 {
		t.Fatalf("expected placeholder replacement without extra message, got %d messages", len(got.messages))
	}
	if got.messages[1].Content != "错误: boom" {
		t.Fatalf("expected trailing placeholder to become error, got %q", got.messages[1].Content)
	}
}

func TestClearContextDoesNotReinjectStalePersonaMessage(t *testing.T) {
	m := Model{
		client:      fakeChatClient{},
		persona:     "stale persona",
		apiKeyReady: true,
		messages: []Message{
			{Role: "system", Content: "stale persona"},
			{Role: "user", Content: "hello"},
		},
	}

	updated, _ := m.handleCommand("/clear-context")
	got := updated.(Model)
	if len(got.messages) != 1 {
		t.Fatalf("expected only confirmation message after clear-context, got %d messages", len(got.messages))
	}
	if got.messages[0].Role != "assistant" {
		t.Fatalf("expected confirmation assistant message, got %+v", got.messages[0])
	}
}
