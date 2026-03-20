package tools

import (
	"context"
	"fmt"
	"strings"
)

type EditTool struct {
	write *WriteTool
	read  *ReadTool
}

func NewEditTool() *EditTool {
	return &EditTool{write: NewWriteTool(), read: NewReadTool()}
}

func (t *EditTool) Name() string { return "edit" }

func (t *EditTool) Run(ctx context.Context, input Input) (Result, error) {
	if strings.TrimSpace(input.Old) == "" {
		return Result{}, fmt.Errorf("old text is required")
	}

	current, err := t.read.Run(ctx, input)
	if err != nil {
		return Result{}, err
	}
	if !strings.Contains(current.Output, input.Old) {
		return Result{}, fmt.Errorf("target text not found in %s", input.Path)
	}

	updated := strings.Replace(current.Output, input.Old, input.New, 1)
	if _, err := t.write.Run(ctx, Input{Root: input.Root, Path: input.Path, Content: updated}); err != nil {
		return Result{}, err
	}

	return Result{Tool: t.Name(), Path: input.Path, Output: updated, Changed: true}, nil
}
