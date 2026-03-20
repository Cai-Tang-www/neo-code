package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type WriteTool struct{}

func NewWriteTool() *WriteTool {
	return &WriteTool{}
}

func (t *WriteTool) Name() string { return "write" }

func (t *WriteTool) Run(ctx context.Context, input Input) (Result, error) {
	_ = ctx
	resolved, err := resolvePath(input.Root, input.Path)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return Result{}, fmt.Errorf("create parent for %s: %w", input.Path, err)
	}
	if err := os.WriteFile(resolved, []byte(input.Content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", input.Path, err)
	}
	return Result{Tool: t.Name(), Path: input.Path, Output: input.Content, Changed: true}, nil
}
