package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/llm"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var llmLoadingSpinner = spinner.Spinner{
	Frames: []string{"▰▱▱▱", "▰▰▱▱", "▰▰▰▱", "▰▰▰▰", "▱▰▰▰", "▱▱▰▰", "▱▱▱▰", "▱▱▱▱"},
	FPS:    time.Second / 12,
}

func (m Model) isLLMConfigMode() bool {
	return m.mode == modeLLMProvider || m.mode == modeLLMServer || m.mode == modeLLMLoading || m.mode == modeLLMModel
}

func (m Model) startLLMConfig() (Model, tea.Cmd) {
	providerType := m.cfg.LLM.Provider
	if providerType != "ollama" && providerType != "vllm" {
		providerType = "omarchy"
	}
	m.llmDraft = llmConfigDraft{ProviderType: providerType, ServerURL: m.cfg.LLM.BaseURL, Model: m.cfg.LLM.Model, ProviderIndex: providerIndex(providerType), PreviousInput: m.input.Value()}
	m.input.Reset()
	m.input.Blur()
	m.mode = modeLLMProvider
	m.status = "select llm provider"
	m.err = ""
	m.searching = false
	m.refreshTitle()
	return m, noop
}

func (m Model) handleLLMConfigKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		return m.cancelLLMConfig()
	}
	switch m.mode {
	case modeLLMProvider:
		return m.handleLLMProviderKey(msg)
	case modeLLMServer:
		return m.handleLLMServerKey(msg)
	case modeLLMLoading:
		return m, noop
	case modeLLMModel:
		return m.handleLLMModelKey(msg)
	}
	return m, noop
}

func (m Model) handleLLMProviderKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ProviderIndex = max(0, m.llmDraft.ProviderIndex-1)
	case "down", "right":
		m.llmDraft.ProviderIndex = min(2, m.llmDraft.ProviderIndex+1)
	case "1":
		m.llmDraft.ProviderIndex = 0
	case "2":
		m.llmDraft.ProviderIndex = 1
	case "3":
		m.llmDraft.ProviderIndex = 2
	case "enter":
		m.llmDraft.ProviderType = []string{"omarchy", "ollama", "vllm"}[m.llmDraft.ProviderIndex]
		if m.llmDraft.ProviderType == "omarchy" {
			m.llmDraft.Model = ""
			m.llmDraft.ServerURL = ""
			return m.saveLLMConfig()
		}
		if m.llmDraft.ServerURL == "" || m.cfg.LLM.Provider != m.llmDraft.ProviderType {
			m.llmDraft.ServerURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
		}
		m.input.SetValue(llm.NormalizeServerURL(m.llmDraft.ProviderType, m.llmDraft.ServerURL))
		m.input.CursorEnd()
		m.input.Placeholder = llm.DefaultServerURL(m.llmDraft.ProviderType)
		m.input.Focus()
		m.mode = modeLLMServer
		m.status = "set llm server"
	}
	return m, noop
}

func (m Model) handleLLMServerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.String() != "enter" {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	serverURL := llm.NormalizeServerURL(m.llmDraft.ProviderType, m.input.Value())
	if serverURL == "" {
		serverURL = llm.DefaultServerURL(m.llmDraft.ProviderType)
	}
	m.llmDraft.ServerURL = serverURL
	m.llmDraft.Models = nil
	m.llmDraft.FetchErr = ""
	m.llmDraft.ModelIndex = 0
	m.input.Reset()
	m.input.Blur()
	m.mode = modeLLMLoading
	m.status = "querying models"
	m.searching = true
	m.spinner.Spinner = llmLoadingSpinner
	return m, tea.Batch(m.fetchLLMModels(), m.spinner.Tick)
}

func (m Model) fetchLLMModels() tea.Cmd {
	providerType, serverURL := m.llmDraft.ProviderType, m.llmDraft.ServerURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := llm.FetchProviderModels(ctx, providerType, serverURL)
		return llmModelsMsg{models: models, err: err}
	}
}

func (m Model) handleLLMModelsMsg(msg llmModelsMsg) (Model, tea.Cmd) {
	if m.mode != modeLLMLoading {
		return m, nil
	}
	m.searching = false
	m.spinner.Spinner = spinner.Jump
	m.llmDraft.Models = msg.models
	if msg.err != nil {
		m.llmDraft.FetchErr = msg.err.Error()
	} else if len(msg.models) == 0 {
		m.llmDraft.FetchErr = "provider returned no models"
	}
	if m.llmDraft.FetchErr != "" {
		m.input.SetValue(m.llmDraft.Model)
		m.input.CursorEnd()
		m.input.Placeholder = "model name"
		m.input.Focus()
		m.status = "enter model manually"
	} else {
		m.llmDraft.ModelIndex = modelChoiceIndex(msg.models, m.cfg.LLM.Model)
		m.status = "select model"
	}
	m.mode = modeLLMModel
	return m, nil
}

func (m Model) handleLLMModelKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		if msg.String() == "enter" {
			model := strings.TrimSpace(m.input.Value())
			if model == "" {
				m.err = "model name is required"
				return m, noop
			}
			m.llmDraft.Model = model
			return m.saveLLMConfig()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "up", "left":
		m.llmDraft.ModelIndex = max(0, m.llmDraft.ModelIndex-1)
	case "down", "right":
		m.llmDraft.ModelIndex = min(len(m.llmDraft.Models)-1, m.llmDraft.ModelIndex+1)
	case "enter":
		m.llmDraft.Model = m.llmDraft.Models[m.llmDraft.ModelIndex]
		return m.saveLLMConfig()
	default:
		if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(m.llmDraft.Models) {
			m.llmDraft.ModelIndex = n - 1
		}
	}
	return m, noop
}

func (m Model) saveLLMConfig() (Model, tea.Cmd) {
	chatPath, modelsPath := llm.ProviderPaths(m.llmDraft.ProviderType)
	m.cfg.LLM = config.LLMConfig{Provider: m.llmDraft.ProviderType}
	if m.llmDraft.ProviderType != "omarchy" {
		m.cfg.LLM.BaseURL = llm.NormalizeServerURL(m.llmDraft.ProviderType, m.llmDraft.ServerURL)
		m.cfg.LLM.Model = m.llmDraft.Model
		m.cfg.LLM.ChatPath = chatPath
		m.cfg.LLM.ModelsPath = modelsPath
	}
	if err := config.Save(m.cfg); err != nil {
		m.err = err.Error()
		return m, noop
	}
	provider, model := m.llmDraft.ProviderType, m.llmDraft.Model
	m.restoreLLMInput()
	m, _ = m.back()
	m.status = strings.TrimSpace(fmt.Sprintf("llm set: %s %s", provider, model))
	m.err = ""
	return m, noop
}

func (m Model) cancelLLMConfig() (Model, tea.Cmd) {
	m.restoreLLMInput()
	m, _ = m.back()
	m.status = "llm config canceled"
	m.err = ""
	return m, noop
}

func (m *Model) restoreLLMInput() {
	previous := m.llmDraft.PreviousInput
	m.llmDraft = llmConfigDraft{}
	m.input.SetValue(previous)
	m.input.CursorEnd()
	m.input.Placeholder = "song, artist, or album"
	m.input.Blur()
	m.searching = false
	m.spinner.Spinner = spinner.Jump
}

func providerIndex(providerType string) int {
	if providerType == "omarchy" {
		return 0
	}
	if providerType == "ollama" {
		return 1
	}
	return 2
}

func modelChoiceIndex(models []string, current string) int {
	for i, model := range models {
		if model == current {
			return i
		}
	}
	return 0
}
