package tui

import (
	"github.com/bprendie/subweazl/internal/audio"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) drainMeter() {
	if m.meter == nil {
		m.energy = audio.Sample{}
		return
	}
	for {
		select {
		case sample, ok := <-m.meter.Samples():
			if !ok {
				if err := m.meterError(); err != nil {
					m.err = "visualizer: " + err.Error()
				}
				m.meter = nil
				m.energy = audio.Sample{}
				return
			}
			m.energy = sample
		default:
			if err := m.meterError(); err != nil {
				m.err = "visualizer: " + err.Error()
				m.meter = nil
				m.energy = audio.Sample{}
			}
			return
		}
	}
}

func (m *Model) meterError() error {
	if m.meter == nil {
		return nil
	}
	select {
	case err, ok := <-m.meter.Errors():
		if !ok {
			return nil
		}
		return err
	default:
		return nil
	}
}

func (m *Model) resize(width, height int) {
	m.width = width
	m.height = height
	contentWidth := max(20, width-4)
	mainWidth := contentWidth
	if contentWidth >= 64 {
		sidebarWidth := clampInt(contentWidth/4, 20, 30)
		mainWidth = max(24, contentWidth-sidebarWidth-6)
	}
	header := m.appHeader(contentWidth)
	footer := m.footer(contentWidth)
	innerHeight := max(6, height-2)
	bodyRenderedHeight := innerHeight - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	bodyRenderedHeight = max(4, bodyRenderedHeight)
	bodyHeight := max(2, bodyRenderedHeight-2)
	m.list.SetSize(mainWidth, bodyHeight)
	m.input.Width = max(20, contentWidth-10)
}

func (m Model) updateFocused(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == " " {
		return m, nil
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func noop() tea.Msg { return nil }

func (m *Model) resetInput() {
	m.input.SetValue("")
	m.input.Prompt = searchPrompt
	m.input.EchoMode = textinput.EchoNormal
	m.input.Blur()
}
