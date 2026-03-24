package tools

import (
	"fmt"
	"sync"

	"go-llm-demo/internal/server/domain"
)

var (
	securityCheckerMu sync.RWMutex
	securityChecker   domain.SecurityChecker
)

// SetSecurityChecker 设置工具执行前使用的安全检查器。
// 传入 nil 表示关闭安全检查（默认行为）。
func SetSecurityChecker(checker domain.SecurityChecker) {
	securityCheckerMu.Lock()
	securityChecker = checker
	securityCheckerMu.Unlock()
}

func getSecurityChecker() domain.SecurityChecker {
	securityCheckerMu.RLock()
	checker := securityChecker
	securityCheckerMu.RUnlock()
	return checker
}

func guardToolExecution(toolType, target, toolName string) *ToolResult {
	checker := getSecurityChecker()
	if checker == nil {
		return nil
	}

	action := checker.Check(toolType, target)
	metadata := map[string]interface{}{
		"securityToolType": toolType,
		"securityTarget":   target,
		"securityAction":   string(action),
	}

	switch action {
	case domain.ActionAllow:
		return nil
	case domain.ActionDeny:
		return &ToolResult{
			ToolName: toolName,
			Success:  false,
			Error:    fmt.Sprintf("安全策略拒绝执行 %s: %s", toolType, target),
			Metadata: metadata,
		}
	case domain.ActionAsk:
		return &ToolResult{
			ToolName: toolName,
			Success:  false,
			Error:    fmt.Sprintf("命中安全策略，执行 %s 前需要用户确认: %s", toolType, target),
			Metadata: metadata,
		}
	default:
		metadata["securityAction"] = string(domain.ActionAsk)
		return &ToolResult{
			ToolName: toolName,
			Success:  false,
			Error:    fmt.Sprintf("安全策略返回未知动作(%s)，已按需确认处理: %s", action, target),
			Metadata: metadata,
		}
	}
}
