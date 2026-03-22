package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"unicode/utf8"

	"go-llm-demo/configs"
	"go-llm-demo/internal/server/domain"
	"go-llm-demo/internal/server/infra/provider"
	servertools "go-llm-demo/internal/server/infra/tools"
	"go-llm-demo/internal/tui/infra"

	tea "github.com/charmbracelet/bubbletea"
)

const maxToolSteps = 8

// Update 处理 Bubble Tea 事件并驱动聊天状态更新。
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case StreamChunkMsg:
		if m.generating {
			m.AppendLastMessage(msg.Content)
		}
		return m, waitForAgentEvent(m.agentEvents)
	case StreamDoneMsg:
		m.generating = false
		m.toolExecuting = false
		m.MarkLastMessageStreaming(false)
		return m, nil
	case StreamErrorMsg:
		m.generating = false
		m.toolExecuting = false
		m.MarkLastMessageStreaming(false)
		m.AddMessage("assistant", fmt.Sprintf("错误: %v", msg.Err))
		m.TrimHistory(m.historyTurns)
		return m, nil
	case AgentEventMsg:
		if msg.Event.Message != "" {
			m.AddMessage("system", msg.Event.Message)
		}
		return m, waitForAgentEvent(m.agentEvents)
	case ToolCallDetectedMsg:
		m.toolExecuting = true
		m.generating = false
		m.MarkLastMessageStreaming(false)
		m.AddMessage("system", fmt.Sprintf("检测到工具调用: %s", msg.Call.Tool))
		return m, waitForAgentEvent(m.agentEvents)
	case ToolExecutionStartMsg:
		m.toolExecuting = true
		m.AddMessage("system", formatToolProgress(msg.Call))
		return m, waitForAgentEvent(m.agentEvents)
	case ToolResultMsg:
		m.toolExecuting = false
		m.AddMessage("system", fmt.Sprintf("工具执行结果(%s): %s", msg.Result.ToolName, strings.TrimSpace(msg.Result.Output)))
		m.AddMessage("assistant", "")
		m.MarkLastMessageStreaming(true)
		m.generating = true
		return m, waitForAgentEvent(m.agentEvents)
	case ToolErrorMsg:
		m.toolExecuting = false
		m.AddMessage("system", fmt.Sprintf("工具执行错误: %v", msg.Err))
		m.AddMessage("assistant", "")
		m.MarkLastMessageStreaming(true)
		m.generating = true
		return m, waitForAgentEvent(m.agentEvents)
	case ShowHelpMsg:
		m.mode = ModeHelp
		return m, nil
	case HideHelpMsg:
		m.mode = ModeChat
		return m, nil
	case RefreshMemoryMsg:
		stats, err := m.client.GetMemoryStats(context.Background())
		if err == nil && stats != nil {
			m.memoryStats = *stats
		}
		return m, nil
	case ExitMsg:
		return m, tea.Quit
	}

	return m, cmd
}

func waitForAgentEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return StreamDoneMsg{}
		}
		return msg
	}
}

func (m *Model) emitAgentMsg(msg tea.Msg) {
	m.agentEvents <- msg
}

func (m *Model) startAgentLoop(messages []infra.Message) tea.Cmd {
	go m.runAgentLoop(messages)
	return waitForAgentEvent(m.agentEvents)
}

func (m *Model) runAgentLoop(initial []infra.Message) {
	messages := append([]infra.Message{}, initial...)
	for step := 0; step < maxToolSteps; step++ {
		m.emitAgentMsg(AgentEventMsg{Event: domain.AgentEvent{Type: "agent.thought", Message: fmt.Sprintf("第 %d 步：请求模型响应", step+1)}})
		stream, err := m.client.Chat(context.Background(), messages, m.activeModel)
		if err != nil {
			m.emitAgentMsg(StreamErrorMsg{Err: err})
			return
		}

		var replyBuilder strings.Builder
		for chunk := range stream {
			replyBuilder.WriteString(chunk)
			m.emitAgentMsg(StreamChunkMsg{Content: chunk})
		}

		reply := strings.TrimSpace(replyBuilder.String())
		call, ok, err := extractToolCall(reply)
		if err != nil {
			m.emitAgentMsg(AgentEventMsg{Event: domain.AgentEvent{Type: "tool.parse.warning", Message: fmt.Sprintf("工具调用解析失败，按普通回复处理: %v", err)}})
			m.emitAgentMsg(StreamDoneMsg{})
			return
		}
		if !ok {
			m.emitAgentMsg(StreamDoneMsg{})
			return
		}

		m.emitAgentMsg(ToolCallDetectedMsg{Call: call})
		m.emitAgentMsg(ToolExecutionStartMsg{Call: call})
		result := servertools.GlobalRegistry.Execute(call)
		observation := ""
		if result.Success {
			m.emitAgentMsg(ToolResultMsg{Result: result})
			observation = fmt.Sprintf("工具执行结果(%s): %s", result.ToolName, strings.TrimSpace(result.Output))
		} else {
			errMsg := fmt.Errorf("%s", result.Error)
			m.emitAgentMsg(ToolErrorMsg{Err: errMsg})
			observation = fmt.Sprintf("工具执行错误(%s): %s", result.ToolName, result.Error)
		}

		messages = append(messages, infra.Message{Role: "system", Content: observation}, infra.Message{Role: "assistant", Content: ""})
	}
	m.emitAgentMsg(StreamErrorMsg{Err: fmt.Errorf("工具调用超过最大步数 %d", maxToolSteps)})
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.lastKeyWasEnter = true
		return m.handleNewline()
	case tea.KeyF5, tea.KeyF8:
		return m.handleSubmit()
	case tea.KeyUp:
		if m.multilineMode {
			m.moveCursorUp()
			return *m, nil
		}
		if len(m.commandHistory) > 0 {
			if m.cmdHistIndex < len(m.commandHistory)-1 {
				m.cmdHistIndex++
			}
			if m.cmdHistIndex >= 0 && m.cmdHistIndex < len(m.commandHistory) {
				m.inputBuffer = m.commandHistory[len(m.commandHistory)-1-m.cmdHistIndex]
				m.cursorLine = 0
				m.cursorCol = len(m.inputBuffer)
			}
		}
		return *m, nil
	case tea.KeyDown:
		if m.multilineMode {
			m.moveCursorDown()
			return *m, nil
		}
		if m.cmdHistIndex > 0 {
			m.cmdHistIndex--
			m.inputBuffer = m.commandHistory[len(m.commandHistory)-1-m.cmdHistIndex]
		} else {
			m.cmdHistIndex = -1
			m.inputBuffer = ""
		}
		return *m, nil
	case tea.KeyLeft:
		if m.multilineMode {
			m.moveCursorLeft()
		}
		return *m, nil
	case tea.KeyRight:
		if m.multilineMode {
			m.moveCursorRight()
		}
		return *m, nil
	case tea.KeyHome:
		if m.multilineMode {
			m.cursorCol = 0
		}
		return *m, nil
	case tea.KeyEnd:
		if m.multilineMode {
			lines := strings.Split(m.inputBuffer, "\n")
			if m.cursorLine < len(lines) {
				m.cursorCol = len(lines[m.cursorLine])
			}
		}
		return *m, nil
	case tea.KeyDelete:
		if m.multilineMode {
			m.deleteCharAtCursor()
		}
		return *m, nil
	case tea.KeyTab:
		if m.multilineMode {
			m.insertAtCursor("\t")
		} else {
			m.inputBuffer += "\t"
		}
		return *m, nil
	case tea.KeyRunes:
		if m.lastKeyWasEnter {
			m.lastKeyWasEnter = false
			runes := msg.Runes
			if len(runes) == 1 && runes[0] == 27 {
				return m.handleSubmit()
			}
		}
		pasteField := reflect.ValueOf(msg).FieldByName("Paste")
		isPaste := pasteField.IsValid() && pasteField.Bool()
		r := string(msg.Runes)
		if isPaste && !m.multilineMode && strings.Contains(r, "\n") {
			m.enterMultilineMode()
		}
		if len(r) > 0 && (r[0] >= 32 || r[0] == 9) {
			if m.multilineMode {
				m.insertAtCursor(r)
			} else {
				m.inputBuffer += r
				m.cursorCol++
			}
		} else if len(r) > 0 && r[0] < 32 && r[0] != 9 {
			if r[0] == 10 || r[0] == 13 {
				m.handleNewline()
			}
		}
		m.cmdHistIndex = -1
		return *m, nil
	case tea.KeyBackspace:
		if m.multilineMode {
			m.backspaceAtCursor()
		} else if len(m.inputBuffer) > 0 {
			runes := []rune(m.inputBuffer)
			m.inputBuffer = string(runes[:len(runes)-1])
		}
		return *m, nil
	case tea.KeyEsc:
		if m.mode == ModeHelp {
			m.mode = ModeChat
		}
		return *m, nil
	}
	return *m, nil
}

func (m *Model) handleNewline() (tea.Model, tea.Cmd) {
	if !m.multilineMode {
		m.enterMultilineMode()
	}
	lines := strings.Split(m.inputBuffer, "\n")
	if m.cursorLine < len(lines) {
		line := lines[m.cursorLine]
		runes := []rune(line)
		if m.cursorCol > len(runes) {
			m.cursorCol = len(runes)
		}
		before := string(runes[:m.cursorCol])
		after := string(runes[m.cursorCol:])
		lines[m.cursorLine] = before
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:m.cursorLine+1]...)
		newLines = append(newLines, after)
		if m.cursorLine < len(lines)-1 {
			newLines = append(newLines, lines[m.cursorLine+1:]...)
		}
		m.inputBuffer = strings.Join(newLines, "\n")
	} else {
		m.inputBuffer += "\n"
	}
	m.cursorLine++
	m.cursorCol = 0
	return *m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	m.multilineMode = false
	m.cursorLine = 0
	m.cursorCol = 0
	input := strings.TrimSpace(m.inputBuffer)
	m.inputBuffer = ""
	if input == "" {
		return *m, nil
	}
	if m.mode == ModeHelp {
		m.mode = ModeChat
		return *m, nil
	}
	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}
	if !m.apiKeyReady {
		m.AddMessage("assistant", "当前 API Key 未通过校验，请使用 /apikey <env_name> 切换变量名，或 /exit 退出。")
		return *m, nil
	}
	m.AddMessage("user", input)
	m.AddMessage("assistant", "")
	m.MarkLastMessageStreaming(true)
	m.TrimHistory(m.historyTurns)
	m.generating = true
	m.commandHistory = append(m.commandHistory, input)
	m.cmdHistIndex = -1
	messages := m.buildMessages()
	return *m, m.startAgentLoop(messages)
}

func (m *Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return *m, nil
	}
	cmd := fields[0]
	args := fields[1:]
	if !m.apiKeyReady && !isAPIKeyRecoveryCommand(cmd) {
		m.AddMessage("assistant", "当前 API Key 未通过校验，仅支持 /apikey <env_name>、/help、/models、/switch <model> 或 /exit。")
		return *m, nil
	}
	switch cmd {
	case "/help":
		m.mode = ModeHelp
	case "/exit", "/quit", "/q":
		return *m, tea.Quit
	case "/apikey":
		if len(args) == 0 {
			m.AddMessage("assistant", "用法: /apikey <env_name>")
			return *m, nil
		}
		cfg := configs.GlobalAppConfig
		if cfg == nil {
			m.AddMessage("assistant", "当前配置未加载，无法切换 API Key 环境变量名")
			return *m, nil
		}
		previousEnvName := cfg.AI.APIKey
		cfg.AI.APIKey = strings.TrimSpace(args[0])
		envName := cfg.APIKeyEnvVarName()
		if cfg.RuntimeAPIKey() == "" {
			m.apiKeyReady = false
			m.AddMessage("assistant", fmt.Sprintf("环境变量 %s 未设置。请继续使用 /apikey <env_name> 切换，或 /exit 退出。", envName))
			return *m, nil
		}
		err := provider.ValidateChatAPIKey(context.Background(), cfg)
		if err == nil {
			if writeErr := configs.WriteAppConfig(m.configPath, cfg); writeErr != nil {
				cfg.AI.APIKey = previousEnvName
				m.apiKeyReady = configs.RuntimeAPIKey() != ""
				m.AddMessage("assistant", fmt.Sprintf("切换 API Key 环境变量名失败: %v", writeErr))
				return *m, nil
			}
			m.apiKeyReady = true
			m.AddMessage("assistant", fmt.Sprintf("已切换 API Key 环境变量名为 %s，并通过校验。", envName))
			return *m, nil
		}
		m.apiKeyReady = false
		if errors.Is(err, provider.ErrInvalidAPIKey) {
			m.AddMessage("assistant", fmt.Sprintf("环境变量 %s 中的 API Key 无效：%v。请继续使用 /apikey <env_name> 切换，或 /exit 退出。", envName, err))
			return *m, nil
		}
		m.AddMessage("assistant", fmt.Sprintf("环境变量 %s 的 API Key 未通过校验：%v。请继续使用 /apikey <env_name> 切换，或 /exit 退出。", envName, err))
		return *m, nil
	case "/switch":
		if len(args) == 0 {
			m.AddMessage("assistant", "用法: /switch <model>")
			return *m, nil
		}
		target := args[0]
		if !containsModel(m.client.ListModels(), target) {
			m.AddMessage("assistant", fmt.Sprintf("模型不可用: %s", target))
			return *m, nil
		}
		m.activeModel = target
		m.AddMessage("assistant", fmt.Sprintf("已切换到模型: %s", target))
	case "/models":
		models := m.client.ListModels()
		list := strings.Join(models, "\n  - ")
		m.AddMessage("assistant", fmt.Sprintf("可用模型:\n  - %s", list))
	case "/memory":
		stats, err := m.client.GetMemoryStats(context.Background())
		if err != nil {
			m.AddMessage("assistant", fmt.Sprintf("读取记忆统计失败: %v", err))
			return *m, nil
		}
		m.memoryStats = *stats
		m.AddMessage("assistant", fmt.Sprintf("记忆统计:\n  长期: %d\n  会话: %d\n  总计: %d\n  TopK: %d\n  最小分数: %.2f\n  文件: %s\n  类型: %s", stats.PersistentItems, stats.SessionItems, stats.TotalItems, stats.TopK, stats.MinScore, stats.Path, formatTypeStats(stats.ByType)))
	case "/clear-memory":
		if len(args) == 0 || args[0] != "confirm" {
			m.AddMessage("assistant", "此命令会清空长期记忆。请使用 /clear-memory confirm")
			return *m, nil
		}
		if err := m.client.ClearMemory(context.Background()); err != nil {
			m.AddMessage("assistant", fmt.Sprintf("清空长期记忆失败: %v", err))
			return *m, nil
		}
		stats, _ := m.client.GetMemoryStats(context.Background())
		if stats != nil {
			m.memoryStats = *stats
		}
		m.AddMessage("assistant", "已清空本地长期记忆")
	case "/clear-context":
		if err := m.client.ClearSessionMemory(context.Background()); err != nil {
			m.AddMessage("assistant", fmt.Sprintf("清空会话记忆失败: %v", err))
			return *m, nil
		}
		m.messages = nil
		if m.persona != "" {
			m.messages = append(m.messages, Message{Role: "system", Content: m.persona})
		}
		stats, _ := m.client.GetMemoryStats(context.Background())
		if stats != nil {
			m.memoryStats = *stats
		}
		m.AddMessage("assistant", "已清空当前会话上下文")
	case "/run":
		if len(args) > 0 {
			code := strings.Join(args, " ")
			return *m, tea.Batch(tea.Printf("\n--- 运行代码 ---\n"), runCodeCmd(code))
		}
	case "/explain":
		if len(args) > 0 {
			code := strings.Join(args, " ")
			return *m, m.sendCodeToAI(code)
		}
		return *m, nil
	default:
		m.AddMessage("assistant", fmt.Sprintf("未知命令: %s，输入 /help 查看帮助", cmd))
	}
	return *m, nil
}

func isAPIKeyRecoveryCommand(cmd string) bool {
	switch cmd {
	case "/apikey", "/help", "/models", "/switch", "/exit", "/quit", "/q":
		return true
	default:
		return false
	}
}

func containsModel(models []string, target string) bool {
	for _, model := range models {
		if model == target {
			return true
		}
	}
	return false
}

func formatTypeStats(byType map[string]int) string {
	if len(byType) == 0 {
		return "无"
	}
	ordered := []string{domain.TypeUserPreference, domain.TypeProjectRule, domain.TypeCodeFact, domain.TypeFixRecipe, domain.TypeSessionMemory}
	parts := make([]string, 0, len(byType))
	for _, key := range ordered {
		if count := byType[key]; count > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", key, count))
		}
	}
	if len(parts) == 0 {
		return "无"
	}
	return strings.Join(parts, ", ")
}

func formatToolProgress(call domain.ToolCall) string {
	params := servertools.NormalizeParams(call.Params)
	if filePath, ok := params["filePath"].(string); ok && filePath != "" {
		return fmt.Sprintf("%s: 正在处理 %s", call.Tool, filePath)
	}
	if workdir, ok := params["workdir"].(string); ok && workdir != "" {
		return fmt.Sprintf("%s: 在 %s 中执行工具", call.Tool, workdir)
	}
	return fmt.Sprintf("%s: 正在执行工具...", call.Tool)
}

func (m *Model) buildMessages() []infra.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]infra.Message, 0, len(m.messages))
	for _, msg := range m.messages {
		if msg.Role == "system" {
			result = append(result, infra.Message{Role: msg.Role, Content: msg.Content})
		}
	}
	for _, msg := range m.messages {
		if msg.Role != "system" {
			result = append(result, infra.Message{Role: msg.Role, Content: msg.Content})
		}
	}
	return result
}

func (m *Model) streamResponse(messages []infra.Message) tea.Cmd { return m.startAgentLoop(messages) }

func (m *Model) sendCodeToAI(code string) tea.Cmd {
	prompt := fmt.Sprintf("请解释以下代码：\n```\n%s\n```", code)
	m.AddMessage("user", prompt)
	m.AddMessage("assistant", "")
	m.MarkLastMessageStreaming(true)
	m.TrimHistory(m.historyTurns)
	m.generating = true
	messages := m.buildMessages()
	return m.startAgentLoop(messages)
}

func extractToolCall(content string) (domain.ToolCall, bool, error) {
	candidates := []string{strings.TrimSpace(content)}
	candidates = append(candidates, extractJSONCodeBlocks(content)...)
	if obj := extractFirstJSONObject(content); obj != "" {
		candidates = append(candidates, obj)
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		var call domain.ToolCall
		if err := json.Unmarshal([]byte(candidate), &call); err != nil {
			continue
		}
		if strings.TrimSpace(call.Tool) == "" {
			continue
		}
		call.Params = servertools.NormalizeParams(call.Params)
		return call, true, nil
	}
	if strings.Contains(content, "\"tool\"") {
		return domain.ToolCall{}, false, fmt.Errorf("检测到 tool 字段但未能提取合法 JSON")
	}
	return domain.ToolCall{}, false, nil
}

func extractJSONCodeBlocks(content string) []string {
	parts := strings.Split(content, "```")
	blocks := make([]string, 0)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "json") {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "json"))
		}
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			blocks = append(blocks, trimmed)
		}
	}
	return blocks
}

func extractFirstJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return ""
}

func runCodeCmd(code string) tea.Cmd {
	return func() tea.Msg {
		ext, runner := detectLanguage(code)
		if ext == "" {
			return StreamErrorMsg{Err: fmt.Errorf("无法识别代码语言")}
		}
		tmpFile, err := os.CreateTemp("", "neocode-*."+ext)
		if err != nil {
			return StreamErrorMsg{Err: fmt.Errorf("创建临时文件失败: %w", err)}
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.WriteString(code); err != nil {
			return StreamErrorMsg{Err: fmt.Errorf("写入临时文件失败: %w", err)}
		}
		tmpFile.Close()
		var cmd *exec.Cmd
		if runner != "" {
			cmd = exec.Command(runner, tmpFile.Name())
		} else {
			cmd = exec.Command("go", "run", tmpFile.Name())
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return StreamErrorMsg{Err: err}
		}
		return StreamDoneMsg{}
	}
}

func detectLanguage(code string) (string, string) {
	code = strings.TrimSpace(code)
	if strings.HasPrefix(code, "#!/bin/bash") || strings.HasPrefix(code, "#!/bin/sh") {
		return "sh", "bash"
	}
	if strings.HasPrefix(code, "package main") || strings.Contains(code, "func main()") {
		return "go", ""
	}
	if strings.HasPrefix(code, "def ") || strings.HasPrefix(code, "class ") {
		return "py", "python"
	}
	if strings.HasPrefix(code, "fn ") || strings.HasPrefix(code, "impl ") {
		return "rs", "rustc"
	}
	if strings.HasPrefix(code, "console.log") || strings.Contains(code, "=>") {
		return "js", "node"
	}
	return "", ""
}

func (m *Model) moveCursorUp() {
	if m.cursorLine > 0 {
		m.cursorLine--
		lines := strings.Split(m.inputBuffer, "\n")
		if m.cursorLine < len(lines) {
			lineRunes := utf8.RuneCountInString(lines[m.cursorLine])
			if m.cursorCol > lineRunes {
				m.cursorCol = lineRunes
			}
		}
	}
}
func (m *Model) moveCursorDown() {
	lines := strings.Split(m.inputBuffer, "\n")
	if m.cursorLine < len(lines)-1 {
		m.cursorLine++
		lineRunes := utf8.RuneCountInString(lines[m.cursorLine])
		if m.cursorCol > lineRunes {
			m.cursorCol = lineRunes
		}
	}
}
func (m *Model) moveCursorLeft() {
	if m.cursorLine == 0 && m.cursorCol == 0 {
		return
	}
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorLine > 0 {
		m.cursorLine--
		lines := strings.Split(m.inputBuffer, "\n")
		m.cursorCol = utf8.RuneCountInString(lines[m.cursorLine])
	}
}
func (m *Model) moveCursorRight() {
	lines := strings.Split(m.inputBuffer, "\n")
	currentLineLen := utf8.RuneCountInString(lines[m.cursorLine])
	if m.cursorCol < currentLineLen {
		m.cursorCol++
	} else if m.cursorLine < len(lines)-1 {
		m.cursorLine++
		m.cursorCol = 0
	}
}
func (m *Model) insertAtCursor(text string) {
	lines := strings.Split(m.inputBuffer, "\n")
	if m.cursorLine >= len(lines) {
		lines = append(lines, "")
	}
	lineRunes := []rune(lines[m.cursorLine])
	if m.cursorCol > len(lineRunes) {
		m.cursorCol = len(lineRunes)
	}
	newLine := string(lineRunes[:m.cursorCol]) + text + string(lineRunes[m.cursorCol:])
	lines[m.cursorLine] = newLine
	m.inputBuffer = strings.Join(lines, "\n")
	m.cursorCol += utf8.RuneCountInString(text)
}
func (m *Model) backspaceAtCursor() {
	lines := strings.Split(m.inputBuffer, "\n")
	if m.cursorLine >= len(lines) {
		return
	}
	if m.cursorCol > 0 {
		lineRunes := []rune(lines[m.cursorLine])
		lines[m.cursorLine] = string(lineRunes[:m.cursorCol-1]) + string(lineRunes[m.cursorCol:])
		m.cursorCol--
	} else if m.cursorLine > 0 {
		prevLen := utf8.RuneCountInString(lines[m.cursorLine-1])
		lines[m.cursorLine-1] += lines[m.cursorLine]
		lines = append(lines[:m.cursorLine], lines[m.cursorLine+1:]...)
		m.cursorLine--
		m.cursorCol = prevLen
	}
	m.inputBuffer = strings.Join(lines, "\n")
}
func (m *Model) deleteCharAtCursor() {
	lines := strings.Split(m.inputBuffer, "\n")
	if m.cursorLine >= len(lines) {
		return
	}
	lineRunes := []rune(lines[m.cursorLine])
	if m.cursorCol < len(lineRunes) {
		lines[m.cursorLine] = string(lineRunes[:m.cursorCol]) + string(lineRunes[m.cursorCol+1:])
	} else if m.cursorLine < len(lines)-1 {
		lines[m.cursorLine] += lines[m.cursorLine+1]
		lines = append(lines[:m.cursorLine+1], lines[m.cursorLine+2:]...)
	}
	m.inputBuffer = strings.Join(lines, "\n")
}
func (m *Model) enterMultilineMode() { m.multilineMode = true }
