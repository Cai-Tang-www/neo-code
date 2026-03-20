package tools

import "context"

// Input is a generic request envelope for the minimal toolset.
type Input struct {
	Root    string
	Path    string
	Content string
	Old     string
	New     string
	Command string
}

// Result is the common response payload returned by all tools.
type Result struct {
	Tool    string
	Path    string
	Command string
	Output  string
	Changed bool
}

// Tool describes the common behavior for all first-phase tools.
type Tool interface {
	Name() string
	Run(ctx context.Context, input Input) (Result, error)
}
