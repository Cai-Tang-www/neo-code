package agent

import "neo-code/internal/tools"

type Session struct {
	WorkspaceRoot string
	LastResult    *tools.Result
}
