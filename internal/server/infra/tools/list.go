package tools

import (
	"fmt"
	"os"
	"strings"
)

// ListTool 列出目录内容。
type ListTool struct{}

func (l *ListTool) Name() string { return "list" }

func (l *ListTool) Description() string {
	return "列出指定路径中的文件和目录。每行返回一个条目，子目录后跟 '/'。"
}

func (l *ListTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        l.Name(),
		Description: l.Description(),
		Parameters:  []ToolParameter{{Name: "path", Type: "string", Description: "要列出的目录路径"}},
	}
}

func (l *ListTool) Run(params map[string]interface{}) *ToolResult {
	path := "."
	if pathParam, ok := params["path"]; ok {
		var ok2 bool
		path, ok2 = pathParam.(string)
		if !ok2 {
			return &ToolResult{ToolName: l.Name(), Success: false, Error: "path 必须是字符串"}
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return &ToolResult{ToolName: l.Name(), Success: false, Error: fmt.Sprintf("打开目录失败: %v", err)}
	}
	defer file.Close()

	entries, err := file.Readdir(-1)
	if err != nil {
		return &ToolResult{ToolName: l.Name(), Success: false, Error: fmt.Sprintf("读取目录失败: %v", err)}
	}

	var output strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			output.WriteString(entry.Name() + "/\n")
		} else {
			output.WriteString(entry.Name() + "\n")
		}
	}

	return &ToolResult{ToolName: l.Name(), Success: true, Output: output.String(), Metadata: map[string]interface{}{"path": path, "count": len(entries)}}
}
