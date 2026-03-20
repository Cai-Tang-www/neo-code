package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"neo-code/internal/tools"
	"neo-code/internal/workspace"
)

type Runtime struct {
	workspace *workspace.Manager
	session   *Session
	readTool  tools.Tool
	writeTool tools.Tool
	replTool  tools.Tool
	runTool   tools.Tool
}

func NewRuntime(root string) (*Runtime, error) {
	ws, err := workspace.New(root)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		workspace: ws,
		session: &Session{
			WorkspaceRoot: ws.Root(),
		},
		readTool:  tools.NewReadFileTool(ws),
		writeTool: tools.NewWriteFileTool(ws),
		replTool:  tools.NewReplaceInFileTool(ws),
		runTool:   tools.NewRunCommandTool(ws, 30*time.Second),
	}, nil
}

func (r *Runtime) WorkspaceRoot() string {
	return r.workspace.Root()
}

func (r *Runtime) LastResult() *tools.Result {
	return r.session.LastResult
}

func (r *Runtime) ReadFile(ctx context.Context, path string) (tools.Result, error) {
	return r.execute(ctx, r.readTool, tools.Input{Path: path})
}

func (r *Runtime) WriteFile(ctx context.Context, path, content string) (tools.Result, error) {
	return r.execute(ctx, r.writeTool, tools.Input{Path: path, Content: content})
}

func (r *Runtime) ReplaceInFile(ctx context.Context, path, old, new string) (tools.Result, error) {
	return r.execute(ctx, r.replTool, tools.Input{Path: path, Old: old, New: new})
}

func (r *Runtime) RunCommand(ctx context.Context, command string) (tools.Result, error) {
	return r.execute(ctx, r.runTool, tools.Input{Command: command})
}

func (r *Runtime) execute(ctx context.Context, tool tools.Tool, input tools.Input) (tools.Result, error) {
	result, err := tool.Run(ctx, input)
	if result.Tool != "" {
		r.session.LastResult = &result
	}
	return result, err
}

func HelpText() string {
	return strings.Join([]string{
		"Commands:",
		"  /read <path>                     Read a file from the workspace",
		"  /write <path> <content>          Overwrite a file with inline content",
		"  /replace <path> <old> <new>      Replace the first matching text in a file",
		"  /run <command>                   Run a shell command inside the workspace",
		"  /result                          Show the latest tool result",
		"  /status                          Show the workspace root",
		"  /help                            Show this help text",
		"  /exit                            Exit the program",
		"",
		"Tips:",
		"  * Quote arguments with spaces, e.g. /replace README.md \"old text\" \"new text\"",
		"  * /write expects inline content; use quoted strings for multi-word text.",
	}, "\n")
}

func FormatResult(result tools.Result) string {
	var builder strings.Builder
	builder.WriteString(result.Summary)
	builder.WriteString("\n")
	if result.Path != "" {
		builder.WriteString(fmt.Sprintf("path: %s\n", result.Path))
	}
	if result.Command != "" {
		builder.WriteString(fmt.Sprintf("command: %s\n", result.Command))
	}
	builder.WriteString("output:\n")
	builder.WriteString(result.Output)
	return builder.String()
}
