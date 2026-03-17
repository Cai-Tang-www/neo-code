package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SupportedModels 为所有允许的 ModelScope 模型列表。
var SupportedModels = []string{
	"Qwen/Qwen3-Coder-480B-A35B-Instruct",
	"ZhipuAI/GLM-5",
	"moonshotai/Kimi-K2.5",
	"deepseek-ai/DeepSeek-R1-0528",
}

// DefaultModel 返回默认的模型，当前指向列表中的第一项。
func DefaultModel() string {
	if len(SupportedModels) == 0 {
		return ""
	}
	return SupportedModels[0]
}

// IsSupportedModel 检查模型是否在允许列表中。
func IsSupportedModel(model string) bool {
	for _, m := range SupportedModels {
		if m == model {
			return true
		}
	}
	return false
}

// ModelScopeProvider 是 ModelScope 模型的实现
type ModelScopeProvider struct {
	APIKey string
	Model  string
}

// GetModelName 返回模型名称
func (p *ModelScopeProvider) GetModelName() string {
	return p.Model
}

// StreamResponse 定义 ModelScope 模型的流式返回结构
type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// Chat 实现了 ModelScope 模型的流式对话
func (p *ModelScopeProvider) Chat(ctx context.Context, messages []Message) (<-chan string, error) {
	out := make(chan string)

	go func() {
		defer close(out)
		// 这里调用 API 接口
		url := "https://api-inference.modelscope.cn/v1/chat/completions"
		body := map[string]any{
			"model":    p.Model,
			"messages": messages,
			"stream":   true,
		}
		jsonData, err := json.Marshal(body)
		if err != nil {
			fmt.Println("JSON 编码错误:", err)
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("请求创建错误:", err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("请求发送错误:", err)
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("读取错误:", err)
				return
			}
			line = strings.TrimSpace(line)
			data := strings.TrimPrefix(line, "data: ")
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				break
			}
			var res StreamResponse
			if err := json.Unmarshal([]byte(data), &res); err != nil {
				fmt.Println("JSON 解码错误:", err)
				continue
			}
			if len(res.Choices) > 0 {
				select {
				case <-ctx.Done():
					return
				case out <- res.Choices[0].Delta.Content:
				}
			}
		}

	}()

	return out, nil
}
