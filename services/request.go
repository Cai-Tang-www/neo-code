package services

import (
	"context"
	"neo-code/ai"
)

func Chat(ctx context.Context, messages []ai.Message, model string) (<-chan string, error) {
	provider := &ai.ModelScopeProvider{
		APIKey: "ms-4033c101-6ee1-4619-ba2a-d30496802057",
		Model:  model,
	}
	return provider.Chat(ctx, messages)
}
