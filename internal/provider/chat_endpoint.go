package provider

import (
	"fmt"
	"strings"
)

// ResolveChatEndpointPath 规范化聊天端点路径语义：
// 空字符串或 "/" 代表直连 base_url，其余必须是以 "/" 开头的相对路径。
func ResolveChatEndpointPath(endpointPath string) (string, error) {
	normalizedPath, err := NormalizeProviderChatEndpointPath(endpointPath)
	if err != nil {
		return "", err
	}
	if normalizedPath == "/" {
		return "", nil
	}
	return normalizedPath, nil
}

// ResolveChatEndpointURL 统一根据 base_url 与 chat_endpoint_path 生成聊天请求地址。
// 当 chat_endpoint_path 为空或 "/" 时，按直连模式仅使用 base_url。
func ResolveChatEndpointURL(baseURL string, endpointPath string) (string, error) {
	normalizedPath, err := ResolveChatEndpointPath(endpointPath)
	if err != nil {
		return "", fmt.Errorf("provider chat endpoint path %q is invalid: %w", endpointPath, err)
	}

	normalizedBaseURL, err := NormalizeProviderBaseURL(baseURL)
	if err != nil {
		if normalizedPath == "" {
			return "", fmt.Errorf(
				"provider base_url is invalid for direct chat endpoint mode (chat_endpoint_path is empty or '/'): %w",
				err,
			)
		}
		return "", fmt.Errorf("provider base_url is invalid: %w", err)
	}

	return joinEndpointURL(normalizedBaseURL, normalizedPath), nil
}

// joinEndpointURL 将相对端点路径拼接到已规范化的 base_url。
func joinEndpointURL(baseURL string, endpointPath string) string {
	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	trimmedEndpointPath := strings.TrimSpace(endpointPath)
	if trimmedEndpointPath == "" {
		return trimmedBaseURL
	}
	if !strings.HasPrefix(trimmedEndpointPath, "/") {
		trimmedEndpointPath = "/" + trimmedEndpointPath
	}
	return trimmedBaseURL + trimmedEndpointPath
}
