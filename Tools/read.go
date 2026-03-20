package tools

import (
	"context"
	"fmt"
	"os"
)

type ReadTool struct{}

func NewReadTool() *ReadTool {
	return &ReadTool{}
}

func (t *ReadTool) Name() string { return "read" }

func (t *ReadTool) Run(ctx context.Context, input Input) (Result, error) {
	_ = ctx
	resolved, err := resolvePath(input.Root, input.Path)
	if err != nil {
		return Result{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", input.Path, err)
	}
	return Result{Tool: t.Name(), Path: input.Path, Output: string(data)}, nil
}
