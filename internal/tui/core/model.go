package core

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go-llm-demo/configs"
	"go-llm-demo/internal/tui/infra"
)

type Mode int

const (
	ModeChat Mode = iota
	ModeCodeInput
	ModeHelp
	ModeMemory
)

type Model struct {
	width   int
	height  int
	mode    Mode
	focused string

	messages     []Message
	historyTurns int

	generating  bool
	activeModel string

	memoryStats infra.MemoryStats

	commandHistory []string
	cmdHistIndex   int
	inputBuffer    string

	client          infra.ChatClient
	persona         string
	lastKeyWasEnter bool

	cursorLine    int
	cursorCol     int
	multilineMode bool

	toolExecuting bool
	apiKeyReady   bool
	configPath    string

	agentEvents chan tea.Msg

	mu sync.Mutex
}

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
	Streaming bool
}

var (
	accentStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF")).Bold(true)
	userMsgStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379")).Bold(true)
	assistantMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
	systemMsgStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD"))
	timestampStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
	codeBlockStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#ABB2BF")).Background(lipgloss.Color("#282C34")).Padding(0, 1)
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#61AFEF"))
	dimStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
)

// NewModel 创建 TUI 状态模型。
func NewModel(client infra.ChatClient, persona string, historyTurns int, configPath string) Model {
	stats, _ := client.GetMemoryStats(context.Background())
	if stats == nil {
		stats = &infra.MemoryStats{}
	}
	if historyTurns <= 0 {
		historyTurns = 6
	}
	return Model{
		mode:           ModeChat,
		focused:        "input",
		messages:       make([]Message, 0),
		historyTurns:   historyTurns,
		activeModel:    client.DefaultModel(),
		memoryStats:    *stats,
		commandHistory: make([]string, 0),
		cmdHistIndex:   -1,
		client:         client,
		persona:        persona,
		apiKeyReady:    configs.RuntimeAPIKey() != "",
		configPath:     configPath,
		agentEvents:    make(chan tea.Msg, 128),
	}
}

func (m Model) Init() tea.Cmd    { return nil }
func (m *Model) SetWidth(w int)  { m.width = w }
func (m *Model) SetHeight(h int) { m.height = h }

func (m *Model) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, Message{Role: role, Content: content, Timestamp: time.Now()})
}

func (m *Model) AppendLastMessage(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].Content += content
	}
}

func (m *Model) MarkLastMessageStreaming(streaming bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) > 0 {
		m.messages[len(m.messages)-1].Streaming = streaming
	}
}

func (m *Model) TrimHistory(maxTurns int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) <= maxTurns*2 {
		return
	}
	var system []Message
	var others []Message
	for _, msg := range m.messages {
		if msg.Role == "system" {
			system = append(system, msg)
		} else {
			others = append(others, msg)
		}
	}
	if len(others) > maxTurns*2 {
		others = others[len(others)-maxTurns*2:]
	}
	m.messages = append(system, others...)
}
