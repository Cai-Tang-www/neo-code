package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

type WriteTool struct{}

func (w *WriteTool) Name() string        { return "write" }
func (w *WriteTool) Description() string { return "Write a file to the local filesystem.写入文件" }
func (w *WriteTool) Schema() ToolSchema {
	return ToolSchema{Name: w.Name(), Description: w.Description(), Parameters: []ToolParameter{{Name: "filePath", Type: "string", Required: true, Description: "要写入的文件路径"}, {Name: "content", Type: "string", Required: true, Description: "写入文件的完整内容"}}}
}
func (w *WriteTool) Run(params map[string]interface{}) *ToolResult {
	filePathParam, ok := params["filePath"]
	if !ok {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: "缺少必填参数: filePath"}
	}
	filePath, ok := filePathParam.(string)
	if !ok {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: "filePath 必须是字符串"}
	}
	contentParam, ok := params["content"]
	if !ok {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: "缺少必填参数: content"}
	}
	content, ok := contentParam.(string)
	if !ok {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: "content 必须是字符串"}
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: fmt.Sprintf("创建目录失败: %v", err)}
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return &ToolResult{ToolName: w.Name(), Success: false, Error: fmt.Sprintf("写入文件失败: %v", err)}
	}
	return &ToolResult{ToolName: w.Name(), Success: true, Output: fmt.Sprintf("成功写入 %s", filePath), Metadata: map[string]interface{}{"bytesWritten": len(content), "filePath": filePath}}
}
