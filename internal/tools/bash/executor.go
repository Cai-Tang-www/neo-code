package bash

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"neo-code/internal/security"
	"neo-code/internal/tools"
)

// SecurityExecutor is the bash execution boundary that enforces workspace and
// runtime safety checks before running a shell command.
type SecurityExecutor interface {
	Execute(ctx context.Context, call tools.ToolCallInput, command string, requestedWorkdir string) (tools.ToolResult, error)
}

type commandRunner interface {
	CombinedOutput(ctx context.Context, binary string, args []string, workdir string) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) CombinedOutput(
	ctx context.Context,
	binary string,
	args []string,
	workdir string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = workdir
	return cmd.CombinedOutput()
}

type defaultSecurityExecutor struct {
	root    string
	shell   string
	timeout time.Duration
	runner  commandRunner
}

// NewDefaultSecurityExecutor returns the default secure bash executor.
func NewDefaultSecurityExecutor(root string, shell string, timeout time.Duration) SecurityExecutor {
	return &defaultSecurityExecutor{
		root:    root,
		shell:   shell,
		timeout: timeout,
		runner:  execCommandRunner{},
	}
}

func (e *defaultSecurityExecutor) Execute(
	ctx context.Context,
	call tools.ToolCallInput,
	command string,
	requestedWorkdir string,
) (tools.ToolResult, error) {
	trimmedCommand := strings.TrimSpace(command)
	if trimmedCommand == "" {
		err := errors.New("bash: command is empty")
		return tools.NewErrorResult("bash", tools.NormalizeErrorReason("bash", err), "", nil), err
	}
	if reason, denied := hardDenyBashCommand(trimmedCommand); denied {
		err := fmt.Errorf("%w: %s", tools.ErrPermissionDenied, reason)
		result := tools.NewErrorResult("bash", tools.NormalizeErrorReason("bash", err), reason, nil)
		result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
		return result, err
	}

	base := strings.TrimSpace(call.Workdir)
	if base == "" {
		base = e.root
	}
	_, workdir, err := tools.ResolveWorkspaceTarget(
		call,
		security.TargetTypeDirectory,
		base,
		requestedWorkdir,
		resolveWorkdir,
	)
	if err != nil {
		return tools.NewErrorResult("bash", tools.NormalizeErrorReason("bash", err), "", nil), err
	}

	runCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	binary, args := shellCommand(e.shell, trimmedCommand)
	output, runErr := e.runner.CombinedOutput(runCtx, binary, args, workdir)
	content := string(output)
	if runErr != nil {
		result := tools.NewErrorResult(
			"bash",
			tools.NormalizeErrorReason("bash", runErr),
			content,
			map[string]any{"workdir": workdir},
		)
		result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
		return result, runErr
	}

	result := tools.ToolResult{
		Name:    "bash",
		Content: content,
		Metadata: map[string]any{
			"workdir": workdir,
		},
	}
	result = tools.ApplyOutputLimit(result, tools.DefaultOutputLimitBytes)
	return result, nil
}

// hardDenyBashCommand 基于高风险关键词进行硬拒绝，避免危险命令通过会话记忆被长期放行。
func hardDenyBashCommand(command string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if normalized == "" {
		return "", false
	}

	patterns := map[string]string{
		"rm -rf /":                         "destructive remove on root path is denied",
		"rm -rf /*":                        "destructive remove on root path is denied",
		"sudo ":                            "sudo escalation is denied",
		"mkfs":                             "disk formatting command is denied",
		"dd if=":                           "raw disk write command is denied",
		"shutdown ":                        "system shutdown command is denied",
		"reboot":                           "system reboot command is denied",
		"del /s /q c:\\":                   "destructive remove on system root is denied",
		"format c:":                        "disk format command is denied",
		"remove-item -recurse -force c:\\": "destructive Remove-Item on system root is denied",
		"remove-item -recurse -force /":    "destructive Remove-Item on root path is denied",
	}
	for pattern, reason := range patterns {
		if strings.Contains(normalized, pattern) {
			return reason, true
		}
	}
	return "", false
}

func shellCommand(shell string, command string) (string, []string) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "powershell", "pwsh":
		return "powershell", []string{"-NoProfile", "-Command", command}
	case "bash":
		return "bash", []string{"-lc", command}
	case "sh":
		return "sh", []string{"-lc", command}
	}

	if runtime.GOOS == "windows" {
		return "powershell", []string{"-NoProfile", "-Command", command}
	}
	return "sh", []string{"-lc", command}
}

func resolveWorkdir(root string, requested string) (string, error) {
	if strings.ContainsRune(root, '\x00') || strings.ContainsRune(requested, '\x00') {
		return "", errors.New("bash: invalid path contains NUL")
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := requested
	if strings.TrimSpace(target) == "" {
		target = base
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(base, target)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("bash: workdir escapes workspace root")
	}
	return target, nil
}
