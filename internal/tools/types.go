package tools

import "context"

type Result struct {
	Tool    string
	Summary string
	Output  string
	Path    string
	Command string
	Changed bool
}

type Tool interface {
	Name() string
	Run(ctx context.Context, input Input) (Result, error)
}

type Input struct {
	Path    string
	Content string
	Old     string
	New     string
	Command string
}
