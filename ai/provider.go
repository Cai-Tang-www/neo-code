package ai

import (
	"context"
)

type Provider interface {
	GetModelName() string
	Chat(ctx context.Context, messages []Message) (<-chan string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
