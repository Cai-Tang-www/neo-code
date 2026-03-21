package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Tool 定义了所有工具必须实现的接口。
type Tool interface {
	// Name 返回工具的唯一名称。
	Name() string
	// Description 返回工具的人类可读描述。
	Description() string
	// Schema 返回工具调用参数的结构化定义。
	Schema() ToolSchema
	// Run 执行工具并返回 ToolResult。
	Run(params map[string]interface{}) *ToolResult
}

// ToolSchema 描述工具参数结构。
type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`
}

// ToolParameter 描述单个工具参数。
type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToolResult 表示执行工具的结果。
type ToolResult struct {
	ToolName string                 `json:"tool"`
	Success  bool                   `json:"success"`
	Output   string                 `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolRegistry 管理工具的注册和检索。
type ToolRegistry struct {
	tools map[string]Tool
}

var (
	initToolsOnce sync.Once

	// GlobalRegistry 是 ToolRegistry 的单例实例。
	GlobalRegistry = &ToolRegistry{tools: make(map[string]Tool)}
)

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) Tool {
	return r.tools[name]
}

func (r *ToolRegistry) ListTools() []string {
	keys := make([]string, 0, len(r.tools))
	for k := range r.tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (r *ToolRegistry) Schemas() []ToolSchema {
	names := r.ListTools()
	schemas := make([]ToolSchema, 0, len(names))
	for _, name := range names {
		tool := r.tools[name]
		if tool == nil {
			continue
		}
		schemas = append(schemas, tool.Schema())
	}
	return schemas
}

func ValidateParams(schema ToolSchema, params map[string]interface{}) error {
	allowed := make(map[string]ToolParameter, len(schema.Parameters))
	for _, param := range schema.Parameters {
		allowed[param.Name] = param
		if param.Required {
			value, ok := params[param.Name]
			if !ok || value == nil {
				return fmt.Errorf("缺少必需参数: %s", param.Name)
			}
		}
	}

	for key, value := range params {
		param, ok := allowed[key]
		if !ok {
			return fmt.Errorf("未知参数: %s", key)
		}
		if value == nil {
			continue
		}
		if err := validateParamType(param.Type, value); err != nil {
			return fmt.Errorf("参数 %s 校验失败: %w", key, err)
		}
	}
	return nil
}

func validateParamType(expected string, value interface{}) error {
	switch expected {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("期望 string")
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64, string:
			return nil
		default:
			return fmt.Errorf("期望 number")
		}
	case "boolean":
		switch value.(type) {
		case bool, string:
			return nil
		default:
			return fmt.Errorf("期望 boolean")
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("期望 object")
		}
	}
	return nil
}

func (tr *ToolResult) MarshalJSON() ([]byte, error) {
	type Alias ToolResult
	return json.Marshal(&struct {
		*Alias
		Output string `json:"output,omitempty"`
		Error  string `json:"error,omitempty"`
	}{
		Alias:  (*Alias)(tr),
		Output: tr.Output,
		Error:  tr.Error,
	})
}

func JsonMarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func FormatSchemasForPrompt(schemas []ToolSchema) string {
	var b strings.Builder
	for _, schema := range schemas {
		b.WriteString("- ")
		b.WriteString(schema.Name)
		b.WriteString(": ")
		b.WriteString(schema.Description)
		b.WriteString("\n")
		for _, param := range schema.Parameters {
			required := "optional"
			if param.Required {
				required = "required"
			}
			b.WriteString("  - ")
			b.WriteString(param.Name)
			b.WriteString(" (")
			b.WriteString(param.Type)
			b.WriteString(", ")
			b.WriteString(required)
			b.WriteString("): ")
			b.WriteString(param.Description)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func Initialize() {
	initToolsOnce.Do(func() {
		GlobalRegistry.Register(&ReadTool{})
		GlobalRegistry.Register(&WriteTool{})
		GlobalRegistry.Register(&EditTool{})
		GlobalRegistry.Register(&BashTool{})
		GlobalRegistry.Register(&ListTool{})
		GlobalRegistry.Register(&GrepTool{})
	})
}
