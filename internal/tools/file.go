package tools

import (
	"context"
	"fmt"

	"neo-code/internal/workspace"
)

type ReadFileTool struct {
	workspace *workspace.Manager
}

func NewReadFileTool(workspace *workspace.Manager) *ReadFileTool {
	return &ReadFileTool{workspace: workspace}
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Run(ctx context.Context, input Input) (Result, error) {
	_ = ctx
	content, err := t.workspace.ReadFile(input.Path)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Tool:    t.Name(),
		Path:    input.Path,
		Summary: fmt.Sprintf("Read %s", input.Path),
		Output:  content,
	}, nil
}

type WriteFileTool struct {
	workspace *workspace.Manager
}

func NewWriteFileTool(workspace *workspace.Manager) *WriteFileTool {
	return &WriteFileTool{workspace: workspace}
}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Run(ctx context.Context, input Input) (Result, error) {
	_ = ctx
	if err := t.workspace.WriteFile(input.Path, input.Content); err != nil {
		return Result{}, err
	}
	return Result{
		Tool:    t.Name(),
		Path:    input.Path,
		Changed: true,
		Summary: fmt.Sprintf("Wrote %d bytes to %s", len(input.Content), input.Path),
		Output:  input.Content,
	}, nil
}

type ReplaceInFileTool struct {
	workspace *workspace.Manager
}

func NewReplaceInFileTool(workspace *workspace.Manager) *ReplaceInFileTool {
	return &ReplaceInFileTool{workspace: workspace}
}

func (t *ReplaceInFileTool) Name() string { return "replace_in_file" }

func (t *ReplaceInFileTool) Run(ctx context.Context, input Input) (Result, error) {
	_ = ctx
	updated, err := t.workspace.ReplaceInFile(input.Path, input.Old, input.New)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Tool:    t.Name(),
		Path:    input.Path,
		Changed: true,
		Summary: fmt.Sprintf("Replaced text in %s", input.Path),
		Output:  updated,
	}, nil
}
