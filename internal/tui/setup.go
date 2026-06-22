package tui

import (
	"context"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/subsonic"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	setupServer = iota
	setupUser
	setupPassword
)

func (m Model) handleSetupKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		m.focusSetup((m.setupFocus + 1) % len(m.setup))
		return m, noop
	case "shift+tab", "up":
		m.focusSetup((m.setupFocus - 1 + len(m.setup)) % len(m.setup))
		return m, noop
	case "ctrl+s":
		return m, m.saveSetup(false)
	case "enter", "ctrl+t":
		return m, m.saveSetup(true)
	}
	var cmd tea.Cmd
	m.setup[m.setupFocus], cmd = m.setup[m.setupFocus].Update(msg)
	return m, cmd
}

func (m *Model) focusSetup(next int) {
	for i := range m.setup {
		if i == next {
			m.setup[i].Focus()
			continue
		}
		m.setup[i].Blur()
	}
	m.setupFocus = next
}

func (m Model) setupConfig() config.Config {
	cfg := m.cfg
	cfg.Server = m.setup[setupServer].Value()
	cfg.Username = m.setup[setupUser].Value()
	cfg.Password = m.setup[setupPassword].Value()
	return cfg
}

func (m Model) saveSetup(test bool) tea.Cmd {
	cfg := m.setupConfig()
	return func() tea.Msg {
		if test && cfg.Ready() {
			client := subsonic.New(cfg.Server, cfg.Username, cfg.Password)
			if err := client.Ping(context.Background()); err != nil {
				return errMsg{err}
			}
		}
		if err := config.Save(cfg); err != nil {
			return errMsg{err}
		}
		status := "saved setup"
		if test && cfg.Ready() {
			status = "connected and saved setup"
		}
		return setupSavedMsg{cfg: cfg, status: status}
	}
}
