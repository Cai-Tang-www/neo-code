package domain

import "encoding/json"

type Tool interface {
	Definition() ToolDefinition
	Run(params map[string]interface{}) *ToolResult
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParamSpec `json:"parameters,omitempty"`
}

type ToolParamSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
}

type ToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type AgentEvent struct {
	Type    string                 `json:"type"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// ToolResult 表示执行工具的结果。
type ToolResult struct {
	ToolName string                 `json:"tool"`
	Success  bool                   `json:"success"`
	Output   string                 `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
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
