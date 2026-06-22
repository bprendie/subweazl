package tui

import (
	"github.com/bprendie/subweazl/internal/audio"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
	footerHeight := 7
	if contentWidth < 64 {
		footerHeight = 5
	} else {
		footerHeight = 11
	}
	listHeight := height - 2 - 3 - footerHeight - 2 - 2
	m.list.SetSize(mainWidth, max(2, listHeight))
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
