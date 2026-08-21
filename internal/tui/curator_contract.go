package tui

import (
	"fmt"
	"strings"

	"github.com/bprendie/subweazl/internal/curator"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type curatorDestination int

const (
	curatorDestinationServer curatorDestination = iota
	curatorDestinationVault
)

type curatorRequest struct {
	Mode         curator.Mode
	Destination  curatorDestination
	PlaylistName string
	Limit        int
	Prompt       string
	Seed         subsonic.Track
}

type curatorDraft struct {
	Choice        int
	PreviousInput string
	Request       curatorRequest
}

func moodCuratorRequest(seed subsonic.Track) curatorRequest {
	return curatorRequest{Mode: curator.ModeMood, Destination: curatorDestinationServer, PlaylistName: "Mood", Limit: 20, Seed: seed}
}

func grindageCuratorRequest() curatorRequest {
	return curatorRequest{Mode: curator.ModeAIMix, Destination: curatorDestinationVault, PlaylistName: "zero_tax_grindage", Limit: 40}
}

func promptCuratorRequest(prompt string) curatorRequest {
	return curatorRequest{Mode: curator.ModeAIMix, Destination: curatorDestinationVault, Limit: 40, Prompt: strings.TrimSpace(prompt)}
}

func (m Model) isCuratorInputMode() bool {
	return m.mode == modeCuratorChoice || m.mode == modeCuratorPrompt
}

func (m Model) startCuratorChoice() (Model, tea.Cmd) {
	m.pushNav()
	m.curatorDraft = curatorDraft{PreviousInput: m.input.Value()}
	m.input.Reset()
	m.input.Blur()
	m.mode = modeCuratorChoice
	m.status = "choose a Weazl curator"
	m.err = ""
	m.refreshTitle()
	return m, noop
}

func (m Model) handleCuratorInputKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m.cancelCuratorInput("curator canceled")
	}
	if m.mode == modeCuratorPrompt {
		if msg.String() == "enter" {
			request := promptCuratorRequest(m.input.Value())
			if request.Prompt == "" {
				m.err = "tell Weazl what you want"
				return m, noop
			}
			m.curatorDraft.Request = request
			return m.finishCuratorPhaseZero("request captured; prompt curation lands in phase 1")
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "up", "k", "left":
		m.curatorDraft.Choice = max(0, m.curatorDraft.Choice-1)
	case "down", "j", "right":
		m.curatorDraft.Choice = min(1, m.curatorDraft.Choice+1)
	case "1":
		m.curatorDraft.Choice = 0
	case "2":
		m.curatorDraft.Choice = 1
	case "enter":
		if m.curatorDraft.Choice == 0 {
			m.curatorDraft.Request = grindageCuratorRequest()
			return m.finishCuratorPhaseZero("zero_tax_grindage selected; curation lands in phase 1")
		}
		m.mode = modeCuratorPrompt
		m.input.Prompt = "tell Weazl > "
		m.input.Placeholder = "synthwave tracks for focus"
		m.input.SetValue("")
		m.input.Focus()
		m.status = "tell Weazl what you want"
		m.refreshTitle()
	}
	return m, noop
}

func (m Model) finishCuratorPhaseZero(status string) (Model, tea.Cmd) {
	request := m.curatorDraft.Request
	m.restoreCuratorInput()
	m, _ = m.back()
	m.curatorDraft.Request = request
	m.status = status
	m.err = ""
	return m, noop
}

func (m Model) cancelCuratorInput(status string) (Model, tea.Cmd) {
	m.restoreCuratorInput()
	m, _ = m.back()
	m.status = status
	m.err = ""
	return m, noop
}

func (m *Model) restoreCuratorInput() {
	previous := m.curatorDraft.PreviousInput
	m.input.SetValue(previous)
	m.input.CursorEnd()
	m.input.Prompt = searchPrompt
	m.input.Placeholder = "song, artist, or album"
	m.input.Blur()
}

func (m Model) curatorInputView(width int) string {
	title := lipgloss.NewStyle().Foreground(crushGold).Bold(true).Render("weazl curator")
	help := "[up/down] select  [enter] continue  [esc] cancel"
	body := m.curatorChoiceView()
	if m.mode == modeCuratorPrompt {
		help = "[enter] capture request  [esc] cancel"
		body = lipgloss.JoinVertical(lipgloss.Left, m.styles.help.Render("Describe the crate. Weazl handles the judgment."), "", m.input.View())
	}
	panel := lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Foreground(crushMint).Bold(true).Render(m.list.Title), "", body)
	view := lipgloss.JoinVertical(lipgloss.Left, title, m.styles.help.Render(help), "", m.styles.panel.Width(width).Render(panel), m.statusLine())
	if m.err != "" {
		view += "\n" + m.styles.error.Render(ansi.Wordwrap(m.err, max(20, width-2), " /_-"))
	}
	return view
}

func (m Model) curatorChoiceView() string {
	choices := []struct{ label, desc string }{
		{"zero_tax_grindage", "40-track private crate; no queue mutation"},
		{"Tell Weazl what you want", "prompt-directed private crate"},
	}
	rows := make([]string, 0, len(choices))
	for i, choice := range choices {
		rows = append(rows, m.llmSelectRow(i == m.curatorDraft.Choice, fmt.Sprintf("%d", i+1), choice.label, choice.desc))
	}
	return strings.Join(rows, "\n")
}
