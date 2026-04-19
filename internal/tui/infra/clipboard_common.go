package infra

import (
	"errors"
	"os"
	"strings"
	"sync"
)

var errClipboardImageUnsupported = errors.New("clipboard image is not supported on this platform")

var clipboardHookState struct {
	mu       sync.RWMutex
	copyText func(string) error
}

// runCopyTextHook 在配置了测试钩子时短路真实剪贴板写入，避免测试环境阻塞。
func runCopyTextHook(text string) (handled bool, err error) {
	clipboardHookState.mu.RLock()
	hook := clipboardHookState.copyText
	clipboardHookState.mu.RUnlock()
	if hook == nil {
		return false, nil
	}
	return true, hook(text)
}

// setCopyTextHookForTest 设置测试钩子并返回还原函数，仅供测试用例注入行为。
func setCopyTextHookForTest(hook func(string) error) func() {
	clipboardHookState.mu.Lock()
	prev := clipboardHookState.copyText
	clipboardHookState.copyText = hook
	clipboardHookState.mu.Unlock()
	return func() {
		clipboardHookState.mu.Lock()
		clipboardHookState.copyText = prev
		clipboardHookState.mu.Unlock()
	}
}

func SaveImageToTempFile(data []byte, prefix string) (string, error) {
	pattern := "image-*.png"
	if cleaned := sanitizeTempPrefix(prefix); cleaned != "" {
		pattern = cleaned + "-*.png"
	}

	tempDir := strings.TrimSpace(os.Getenv("TMPDIR"))
	f, err := os.CreateTemp(tempDir, pattern)
	if err != nil {
		return "", err
	}
	tmpFile := f.Name()
	_ = f.Close()
	if err = os.WriteFile(tmpFile, data, 0o600); err != nil {
		_ = os.Remove(tmpFile)
		return "", err
	}

	return tmpFile, nil
}

// sanitizeTempPrefix 过滤临时文件名前缀中的不安全字符，避免路径注入与非法命名。
func sanitizeTempPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	buf := make([]rune, 0, len(prefix))
	for _, r := range prefix {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			buf = append(buf, r)
		}
	}
	return string(buf)
}
