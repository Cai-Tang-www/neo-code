package tools

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadTool 读取文件内容，支持可选的行范围参数。
type ReadTool struct{}

func (r *ReadTool) Name() string { return "read" }

func (r *ReadTool) Description() string {
	return "从本地文件系统读取文件或目录。支持读取特定行范围。"
}

func (r *ReadTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        r.Name(),
		Description: r.Description(),
		Parameters: []ToolParameter{
			{Name: "filePath", Type: "string", Required: true, Description: "要读取的文件或目录路径"},
			{Name: "offset", Type: "number", Description: "开始读取的行号（1-indexed）"},
			{Name: "limit", Type: "number", Description: "最多读取多少行"},
		},
	}
}

func (r *ReadTool) Run(params map[string]interface{}) *ToolResult {
	filePathParam, ok := params["filePath"]
	if !ok {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: "缺少必需参数: filePath"}
	}
	filePath, ok := filePathParam.(string)
	if !ok {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: "filePath 必须是字符串"}
	}

	offset := 1
	if offsetParam, ok := params["offset"]; ok {
		switch v := offsetParam.(type) {
		case float64:
			offset = int(v)
		case int:
			offset = v
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return &ToolResult{ToolName: r.Name(), Success: false, Error: "offset 必须是数字"}
			}
			offset = parsed
		default:
			return &ToolResult{ToolName: r.Name(), Success: false, Error: "offset 必须是数字"}
		}
	}

	limit := 2000
	if limitParam, ok := params["limit"]; ok {
		switch v := limitParam.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		case string:
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return &ToolResult{ToolName: r.Name(), Success: false, Error: "limit 必须是数字"}
			}
			limit = parsed
		default:
			return &ToolResult{ToolName: r.Name(), Success: false, Error: "limit 必须是数字"}
		}
	}

	if offset < 1 {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: "offset 必须 >= 1"}
	}
	if limit < 1 {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: "limit 必须 >= 1"}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: fmt.Sprintf("打开文件失败: %v", err)}
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: fmt.Sprintf("获取文件状态失败: %v", err)}
	}

	var output strings.Builder
	if info.IsDir() {
		files, err := os.ReadDir(filePath)
		if err != nil {
			return &ToolResult{ToolName: r.Name(), Success: false, Error: fmt.Sprintf("读取目录失败: %v", err)}
		}
		for _, f := range files {
			if f.IsDir() {
				output.WriteString(f.Name() + "/\n")
			} else {
				output.WriteString(f.Name() + "\n")
			}
		}
		return &ToolResult{ToolName: r.Name(), Success: true, Output: output.String()}
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	currentLine := 1
	for scanner.Scan() && currentLine < offset {
		currentLine++
	}
	for scanner.Scan() && len(lines) < limit {
		lines = append(lines, scanner.Text())
		currentLine++
	}
	if err := scanner.Err(); err != nil {
		return &ToolResult{ToolName: r.Name(), Success: false, Error: fmt.Sprintf("读取文件错误: %v", err)}
	}

	for i, line := range lines {
		lineNum := offset + i
		output.WriteString(fmt.Sprintf("%d: %s\n", lineNum, line))
	}

	return &ToolResult{ToolName: r.Name(), Success: true, Output: output.String(), Metadata: map[string]interface{}{"linesReturned": len(lines), "offset": offset, "limit": limit}}
}
