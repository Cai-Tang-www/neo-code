package tools

import (
	"strings"
	"testing"

	"go-llm-demo/internal/server/domain"
)

type mockSecurityChecker struct {
	action domain.Action
}

func (m mockSecurityChecker) Check(_ string, _ string) domain.Action {
	return m.action
}

func TestBashTool_Run_DeniedBySecurity(t *testing.T) {
	SetSecurityChecker(mockSecurityChecker{action: domain.ActionDeny})
	defer SetSecurityChecker(nil)

	result := (&BashTool{}).Run(map[string]interface{}{
		"command": "echo hello",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Success {
		t.Fatal("expected bash execution to be denied")
	}
	if !strings.Contains(result.Error, "安全策略拒绝执行") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Metadata["securityAction"] != string(domain.ActionDeny) {
		t.Fatalf("unexpected security action: %#v", result.Metadata["securityAction"])
	}
}

func TestBashTool_Run_AskBySecurity(t *testing.T) {
	SetSecurityChecker(mockSecurityChecker{action: domain.ActionAsk})
	defer SetSecurityChecker(nil)

	result := (&BashTool{}).Run(map[string]interface{}{
		"command": "go build ./...",
	})

	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Success {
		t.Fatal("expected bash execution to require confirmation")
	}
	if !strings.Contains(result.Error, "需要用户确认") {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Metadata["securityAction"] != string(domain.ActionAsk) {
		t.Fatalf("unexpected security action: %#v", result.Metadata["securityAction"])
	}
}
