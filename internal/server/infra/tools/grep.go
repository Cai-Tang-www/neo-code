package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool 使用正则表达式搜索文件内容。
type GrepTool struct{}

func (g *GrepTool) Name() string { return "grep" }

func (g *GrepTool) Description() string {
	return "使用正则表达式搜索文件内容。返回至少有一个匹配项的文件路径和行号。"
}

func (g *GrepTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        g.Name(),
		Description: g.Description(),
		Parameters: []ToolParameter{
			{Name: "pattern", Type: "string", Required: true, Description: "要搜索的正则模式"},
			{Name: "path", Type: "string", Description: "搜索目录"},
			{Name: "include", Type: "string", Description: "文件名 glob 过滤"},
		},
	}
}

func (g *GrepTool) Run(params map[string]interface{}) *ToolResult {
	patternParam, ok := params["pattern"]
	if !ok {
		return &ToolResult{ToolName: g.Name(), Success: false, Error: "缺少必需参数: pattern"}
	}
	pattern, ok := patternParam.(string)
	if !ok {
		return &ToolResult{ToolName: g.Name(), Success: false, Error: "pattern 必须是字符串"}
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return &ToolResult{ToolName: g.Name(), Success: false, Error: fmt.Sprintf("无效的正则表达式模式: %v", err)}
	}

	searchPath := "."
	if pathParam, ok := params["path"]; ok {
		var ok2 bool
		searchPath, ok2 = pathParam.(string)
		if !ok2 {
			return &ToolResult{ToolName: g.Name(), Success: false, Error: "path 必须是字符串"}
		}
	}

	var includePattern string
	if includeParam, ok := params["include"]; ok {
		var ok2 bool
		includePattern, ok2 = includeParam.(string)
		if !ok2 {
			return &ToolResult{ToolName: g.Name(), Success: false, Error: "include 必须是字符串"}
		}
	}

	var results strings.Builder
	var walkErr error
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			walkErr = err
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if includePattern != "" {
			matched, err := filepath.Match(includePattern, filepath.Base(path))
			if err != nil {
				walkErr = err
				return filepath.SkipDir
			}
			if !matched {
				return nil
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		matches := regex.FindAllIndex(content, -1)
		if len(matches) == 0 {
			return nil
		}
		lines := regexp.MustCompile("\\r?\\n").Split(string(content), -1)
		lineCount, lineNum := 0, 0
		for i, line := range lines {
			lineCount += len(line) + 1
			if lineCount > matches[0][0] {
				lineNum = i + 1
				break
			}
		}
		if lineNum > 0 {
			results.WriteString(fmt.Sprintf("%s:%d\n", path, lineNum))
		}
		return nil
	})
	if walkErr != nil {
		return &ToolResult{ToolName: g.Name(), Success: false, Error: walkErr.Error()}
	}
	if err != nil {
		return &ToolResult{ToolName: g.Name(), Success: false, Error: err.Error()}
	}
	if results.Len() == 0 {
		return &ToolResult{ToolName: g.Name(), Success: true, Output: "未找到匹配项。", Metadata: map[string]interface{}{"pattern": pattern, "path": searchPath, "include": includePattern}}
	}
	return &ToolResult{ToolName: g.Name(), Success: true, Output: results.String(), Metadata: map[string]interface{}{"pattern": pattern, "path": searchPath, "include": includePattern, "matches": results.Len()}}
}
