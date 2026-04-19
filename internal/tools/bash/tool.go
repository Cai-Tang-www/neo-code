package bash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"neo-code/internal/tools"
)

type Tool struct {
	root     string
	shell    string
	timeout  time.Duration
	executor SecurityExecutor
}

type input struct {
	Command string `json:"command"`
	Workdir string `json:"workdir,omitempty"`
}

func New(root string, shell string, timeout time.Duration) *Tool {
	executor := NewDefaultSecurityExecutor(root, shell, timeout)
	return &Tool{
		root:     root,
		shell:    shell,
		timeout:  timeout,
		executor: executor,
	}
}

// NewWithExecutor creates a bash tool using an injected security executor.
func NewWithExecutor(root string, shell string, timeout time.Duration, executor SecurityExecutor) *Tool {
	if executor == nil {
		executor = NewDefaultSecurityExecutor(root, shell, timeout)
	}
	return &Tool{
		root:     root,
		shell:    shell,
		timeout:  timeout,
		executor: executor,
	}
}

func (t *Tool) Name() string {
	return tools.ToolNameBash
}

func (t *Tool) Description() string {
	shell := strings.ToLower(strings.TrimSpace(t.shell))
	switch shell {
	case "powershell", "pwsh":
		return "Execute a shell command inside the workspace with timeout and bounded output. " +
			"The command must use PowerShell syntax in this environment."
	case "bash", "sh":
		return fmt.Sprintf(
			"Execute a shell command inside the workspace with timeout and bounded output using %s syntax.",
			shell,
		)
	}
	return "Execute a shell command inside the workspace with timeout and bounded output."
}

func (t *Tool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": t.commandDescription(),
			},
			"workdir": map[string]any{
				"type":        "string",
				"description": "Optional working directory relative to the workspace root.",
			},
		},
		"required": []string{"command"},
	}
}

// commandDescription 生成与当前 shell 对齐的命令描述，减少模型混用命令方言。
func (t *Tool) commandDescription() string {
	shell := strings.ToLower(strings.TrimSpace(t.shell))
	switch shell {
	case "powershell", "pwsh":
		return "PowerShell command to execute. Do not use bash operators like &&, ||, mkdir -p, ls, cat <<EOF."
	case "bash", "sh":
		return fmt.Sprintf("%s command to execute.", shell)
	default:
		return "Shell command to execute."
	}
}

// MicroCompactPolicy 声明 bash 工具的历史结果默认参与 micro compact 清理。
func (t *Tool) MicroCompactPolicy() tools.MicroCompactPolicy {
	return tools.MicroCompactPolicyCompact
}

func (t *Tool) Execute(ctx context.Context, call tools.ToolCallInput) (tools.ToolResult, error) {
	var in input
	if err := json.Unmarshal(call.Arguments, &in); err != nil {
		return tools.NewErrorResult(t.Name(), "invalid arguments", err.Error(), nil), err
	}
	if t.executor == nil {
		err := errors.New("bash: security executor is nil")
		return tools.NewErrorResult(t.Name(), tools.NormalizeErrorReason(t.Name(), err), "", nil), err
	}

	return t.executor.Execute(ctx, call, in.Command, in.Workdir)
}
