package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// BashTool 执行 shell 命令。
type BashTool struct{}

func (b *BashTool) Name() string { return "bash" }

func (b *BashTool) Description() string {
	return "在持久的shell会话中执行给定的bash命令，支持可选超时。"
}

func (b *BashTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        b.Name(),
		Description: b.Description(),
		Parameters: []ToolParameter{
			{Name: "command", Type: "string", Required: true, Description: "要执行的 bash 命令"},
			{Name: "timeout", Type: "number", Description: "超时时间，毫秒"},
			{Name: "workdir", Type: "string", Description: "执行命令的工作目录"},
			{Name: "description", Type: "string", Description: "命令意图说明"},
		},
	}
}

func (b *BashTool) Run(params map[string]interface{}) *ToolResult {
	commandParam, ok := params["command"]
	if !ok {
		return &ToolResult{ToolName: b.Name(), Success: false, Error: "缺少必需参数: command"}
	}
	command, ok := commandParam.(string)
	if !ok {
		return &ToolResult{ToolName: b.Name(), Success: false, Error: "command 必须是字符串"}
	}

	timeoutMs := 120000
	if timeoutParam, ok := params["timeout"]; ok {
		switch v := timeoutParam.(type) {
		case float64:
			timeoutMs = int(v)
		case int:
			timeoutMs = v
		case string:
			parsed, err := parseInt(v)
			if err != nil {
				return &ToolResult{ToolName: b.Name(), Success: false, Error: "timeout 必须是数字"}
			}
			timeoutMs = parsed
		default:
			return &ToolResult{ToolName: b.Name(), Success: false, Error: "timeout 必须是数字"}
		}
	}

	workdir := "."
	if workdirParam, ok := params["workdir"]; ok {
		workdir, ok = workdirParam.(string)
		if !ok {
			return &ToolResult{ToolName: b.Name(), Success: false, Error: "workdir 必须是字符串"}
		}
	}

	ctx, cancel := getContextWithTimeout(timeoutMs)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workdir

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	result := &ToolResult{
		ToolName: b.Name(),
		Metadata: map[string]interface{}{
			"command":   command,
			"workdir":   workdir,
			"timeoutMs": timeoutMs,
		},
	}

	if err != nil {
		if ctx.Err() != nil {
			result.Success = false
			result.Error = fmt.Sprintf("命令在 %dms 后超时", timeoutMs)
		} else {
			result.Success = false
			result.Error = fmt.Sprintf("命令执行失败: %v", err)
			if stderrBuf.Len() > 0 {
				result.Error += fmt.Sprintf(": %s", stderrBuf.String())
			}
		}
		return result
	}

	result.Success = true
	result.Output = stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		result.Output += fmt.Sprintf("\nSTDERR: %s", stderrBuf.String())
	}
	return result
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func getContextWithTimeout(timeoutMs int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
}
