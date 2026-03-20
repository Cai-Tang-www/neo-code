package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type BashTool struct {
	Timeout time.Duration
}

func NewBashTool(timeout time.Duration) *BashTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &BashTool{Timeout: timeout}
}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Run(ctx context.Context, input Input) (Result, error) {
	root, err := resolveRoot(input.Root)
	if err != nil {
		return Result{}, err
	}
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return Result{}, fmt.Errorf("command is required")
	}

	runCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "bash", "-lc", command)
	cmd.Dir = root

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := strings.TrimSpace(strings.Join([]string{stdout.String(), stderr.String()}, "\n"))
	if output == "" {
		output = "(no output)"
	}

	result := Result{Tool: t.Name(), Command: command, Output: output}
	if err != nil {
		return result, fmt.Errorf("run %q: %w", command, err)
	}
	return result, nil
}
