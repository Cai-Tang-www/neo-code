package services

import (
	"context"
	"neo-code/ai"
	"neo-code/config"
)

// Chat 调用 ModelScope 模型进行聊天（封装在Services层）
func Chat(ctx context.Context, messages []ai.Message, model string) (<-chan string, error) {
	provider := &ai.ModelScopeProvider{
		APIKey: config.AppConfig.ModelScopeKey,
		Model:  model,
	}
	return provider.Chat(ctx, messages)
}
