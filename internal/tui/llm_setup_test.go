package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestControlLStartsWeazlLLMWizard(t *testing.T) {
	m := newHomeTestModel(t)
	m.input.Focus()
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	if next.mode != modeLLMProvider || next.llmDraft.ProviderType != "vllm" {
		t.Fatalf("mode=%v draft=%#v", next.mode, next.llmDraft)
	}
	next, _ = next.handleKey(key("enter"))
	if next.mode != modeLLMServer || !next.input.Focused() || next.input.Value() != "http://localhost:8000" {
		t.Fatalf("server step mode=%v value=%q", next.mode, next.input.Value())
	}
	next.input.SetValue("https://granite.prendie.io/v1")
	next, cmd := next.handleKey(key("enter"))
	if next.mode != modeLLMLoading || cmd == nil || next.llmDraft.ServerURL != "https://granite.prendie.io" {
		t.Fatalf("loading step mode=%v url=%q cmd=%v", next.mode, next.llmDraft.ServerURL, cmd)
	}
	next, _ = next.handleLLMModelsMsg(llmModelsMsg{models: []string{"model-a", "model-b"}})
	if next.mode != modeLLMModel || next.llmDraft.ModelIndex != 0 {
		t.Fatalf("model step mode=%v draft=%#v", next.mode, next.llmDraft)
	}
	next, _ = next.handleKey(key("2"))
	next, _ = next.handleKey(key("enter"))
	if next.isLLMConfigMode() || next.cfg.LLM.Provider != "vllm" || next.cfg.LLM.Model != "model-b" || next.cfg.LLM.ChatPath != "/v1/chat/completions" {
		t.Fatalf("saved mode=%v cfg=%#v", next.mode, next.cfg.LLM)
	}
}

func TestLLMWizardSelectsOllamaDefaults(t *testing.T) {
	m := newHomeTestModel(t)
	m.pushNav()
	m, _ = m.startLLMConfig()
	m, _ = m.handleLLMProviderKey(key("2"))
	m, _ = m.handleLLMProviderKey(key("enter"))
	if m.mode != modeLLMServer || m.input.Value() != "http://localhost:11434" {
		t.Fatalf("mode=%v value=%q", m.mode, m.input.Value())
	}
	m.llmDraft.ServerURL = "http://localhost:11434"
	m.llmDraft.Models = []string{"mistral:latest"}
	m.llmDraft.Model = "mistral:latest"
	m.mode = modeLLMModel
	m, _ = m.saveLLMConfig()
	if m.cfg.LLM.Provider != "ollama" || m.cfg.LLM.ChatPath != "/api/chat" || m.cfg.LLM.ModelsPath != "/api/tags" {
		t.Fatalf("cfg=%#v", m.cfg.LLM)
	}
}

func TestLLMWizardEscapeCancels(t *testing.T) {
	m := newHomeTestModel(t)
	original := m.cfg.LLM
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	next, _ = next.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.mode != modeHome || next.cfg.LLM != original || next.status != "llm config canceled" {
		t.Fatalf("mode=%v cfg=%#v status=%q", next.mode, next.cfg.LLM, next.status)
	}
}

func TestHelpPopupTogglesFromMainUI(t *testing.T) {
	m := newHomeTestModel(t)
	next, _ := m.handleKey(key("?"))
	if !next.helpOpen {
		t.Fatal("help popup should open")
	}
	next, _ = next.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.helpOpen {
		t.Fatal("help popup should close")
	}
}

func key(value string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(value)}
}
