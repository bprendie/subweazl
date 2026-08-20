package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) setupView(width int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("subsonic connection")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("[tab] next  [enter] test/save  [ctrl+s] save  [q] quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.panel.Width(width).Render(m.setupPanel(width - 4)))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	if m.err != "" {
		b.WriteString("\n" + m.styles.error.Render(ansi.Wordwrap(m.err, max(20, width-2), " /_-")))
	}
	return b.String()
}

func (m Model) llmConfigView(width int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("llm curator")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render("[up/down] select  [enter] continue  [esc] cancel"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.panel.Width(width).Render(m.llmConfigPanel(width - 4)))
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	if m.err != "" {
		b.WriteString("\n" + m.styles.error.Render(ansi.Wordwrap(m.err, max(20, width-2), " /_-")))
	}
	return b.String()
}

func (m Model) llmConfigPanel(width int) string {
	title := "LLM Provider"
	body := m.providerChoicesView()
	switch m.mode {
	case modeLLMServer:
		title = "LLM Server"
		body = lipgloss.JoinVertical(lipgloss.Left, m.styles.help.Render(serverHint(m.llmDraft.ProviderType)), "", m.input.View())
	case modeLLMLoading:
		title = "LLM Models"
		body = fmt.Sprintf("%s fetching %s models from %s", m.spinner.View(), m.llmDraft.ProviderType, m.llmDraft.ServerURL)
	case modeLLMModel:
		title = "LLM Model"
		body = m.modelChoicesView()
	}
	current := "not configured"
	if m.cfg.LLMReady() {
		current = fmt.Sprintf("%s / %s", m.cfg.LLM.Provider, m.cfg.LLM.Model)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render(title), "", body, "", m.styles.help.Render("current "+current))
}

func (m Model) providerChoicesView() string {
	choices := []string{"vllm", "ollama"}
	rows := make([]string, 0, len(choices))
	for i, choice := range choices {
		rows = append(rows, m.llmSelectRow(i == m.llmDraft.ProviderIndex, fmt.Sprintf("%d", i+1), choice, providerDescription(choice)))
	}
	return strings.Join(rows, "\n")
}

func (m Model) modelChoicesView() string {
	if m.llmDraft.FetchErr != "" || len(m.llmDraft.Models) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, m.styles.error.Render(m.llmDraft.FetchErr), "", m.styles.help.Render("Type the model name exactly as your local server expects."), "", m.input.View())
	}
	rows := make([]string, 0, len(m.llmDraft.Models))
	for i, model := range m.llmDraft.Models {
		rows = append(rows, m.llmSelectRow(i == m.llmDraft.ModelIndex, fmt.Sprintf("%d", i+1), model, ""))
	}
	return strings.Join(rows, "\n")
}

func (m Model) llmSelectRow(selected bool, key, label, desc string) string {
	marker, rowStyle, labelStyle := " ", m.styles.help, m.styles.item
	if selected {
		marker, rowStyle, labelStyle = ">", m.styles.status, m.styles.selected
	}
	left := rowStyle.Render(marker) + " " + lipgloss.NewStyle().Foreground(crushPurple).Bold(true).Render(key) + " " + labelStyle.Render(label)
	if desc == "" {
		return left
	}
	return left + "  " + m.styles.help.Render(desc)
}

func providerDescription(providerType string) string {
	if providerType == "ollama" {
		return "local Ollama /api/chat"
	}
	return "OpenAI-compatible local /v1/chat/completions"
}

func serverHint(providerType string) string {
	if providerType == "ollama" {
		return "Base URL only, without /api. Example: http://localhost:11434"
	}
	return "Base URL only, without /v1. Example: http://localhost:8000"
}

func (m Model) setupPanel(width int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(crushMint).Render("Subsonic / Navidrome"))
	b.WriteString("\n")
	for i := setupServer; i <= setupPassword; i++ {
		b.WriteString(m.setup[i].View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.help.Render(ansi.Truncate("private vault setup follows connection", max(24, width), "...")))
	return b.String()
}
