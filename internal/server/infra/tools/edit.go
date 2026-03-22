package tools

import (
	"fmt"
	"os"
	"strings"
)

// EditTool 在文件中执行精确的字符串替换。
type EditTool struct{}

func (e *EditTool) Name() string { return "edit" }
func (e *EditTool) Description() string {
	return "在文件中执行精确的字符串替换。工具会自动读取文件内容，执行替换操作，并写回文件。"
}
func (e *EditTool) Schema() ToolSchema {
	return ToolSchema{Name: e.Name(), Description: e.Description(), Parameters: []ToolParameter{{Name: "filePath", Type: "string", Required: true, Description: "要修改的文件路径"}, {Name: "oldString", Type: "string", Required: true, Description: "要替换的旧文本"}, {Name: "newString", Type: "string", Required: true, Description: "替换后的新文本"}, {Name: "replaceAll", Type: "boolean", Description: "是否替换全部匹配项"}}}
}
func (e *EditTool) Run(params map[string]interface{}) *ToolResult {
	filePathParam, ok := params["filePath"]
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "缺少必需参数: filePath"}
	}
	filePath, ok := filePathParam.(string)
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "filePath 必须是字符串"}
	}
	oldStringParam, ok := params["oldString"]
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "缺少必需参数: oldString"}
	}
	oldString, ok := oldStringParam.(string)
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "oldString 必须是字符串"}
	}
	newStringParam, ok := params["newString"]
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "缺少必需参数: newString"}
	}
	newString, ok := newStringParam.(string)
	if !ok {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "newString 必须是字符串"}
	}
	if oldString == newString {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: "newString 必须不同于 oldString"}
	}
	replaceAll := false
	if replaceAllParam, ok := params["replaceAll"]; ok {
		switch v := replaceAllParam.(type) {
		case bool:
			replaceAll = v
		case string:
			if v == "true" || v == "1" {
				replaceAll = true
			} else if v == "false" || v == "0" {
				replaceAll = false
			} else {
				return &ToolResult{ToolName: e.Name(), Success: false, Error: "replaceAll 必须是布尔值"}
			}
		default:
			return &ToolResult{ToolName: e.Name(), Success: false, Error: "replaceAll 必须是布尔值"}
		}
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: fmt.Sprintf("读取文件失败: %v", err)}
	}
	fileContent := string(content)
	if !strings.Contains(fileContent, oldString) {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: fmt.Sprintf("未在文件中找到要替换的字符串: %q", oldString)}
	}
	newContent := strings.Replace(fileContent, oldString, newString, 1)
	count := 1
	if replaceAll {
		newContent = strings.ReplaceAll(fileContent, oldString, newString)
		count = strings.Count(fileContent, oldString)
	}
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return &ToolResult{ToolName: e.Name(), Success: false, Error: fmt.Sprintf("写入文件失败: %v", err)}
	}
	return &ToolResult{ToolName: e.Name(), Success: true, Output: fmt.Sprintf("成功替换 %d 处匹配项", count), Metadata: map[string]interface{}{"filePath": filePath, "oldString": oldString, "newString": newString, "replaceAll": replaceAll, "replacements": count, "bytesWritten": len(newContent)}}
}
