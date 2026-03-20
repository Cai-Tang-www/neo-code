package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"neo-code/internal/workspace"
)

type RunCommandTool struct {
	workspace *workspace.Manager
	timeout   time.Duration
}

func NewRunCommandTool(workspace *workspace.Manager, timeout time.Duration) *RunCommandTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &RunCommandTool{workspace: workspace, timeout: timeout}
}

func (t *RunCommandTool) Name() string { return "run_command" }

func (t *RunCommandTool) Run(ctx context.Context, input Input) (Result, error) {
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return Result{}, fmt.Errorf("command is required")
	}

	runCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "bash", "-lc", command)
	cmd.Dir = t.workspace.Root()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errOutput := strings.TrimSpace(stderr.String())
	combined := strings.TrimSpace(strings.Join([]string{output, errOutput}, "\n"))
	if combined == "" {
		combined = "(no output)"
	}

	result := Result{
		Tool:    t.Name(),
		Command: command,
		Summary: fmt.Sprintf("Ran command: %s", command),
		Output:  combined,
	}

	if err != nil {
		return result, fmt.Errorf("run command %q: %w", command, err)
	}

	return result, nil
}
