package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) pushNav() {
	items := append([]list.Item(nil), m.list.Items()...)
	m.nav = append(m.nav, navSnapshot{
		mode:   m.mode,
		items:  items,
		status: m.status,
		cursor: m.list.Index(),
	})
}

func (m *Model) clearNav() {
	m.nav = nil
}

func (m Model) back() (Model, tea.Cmd) {
	if m.input.Focused() {
		m.renaming = nil
		m.resetInput()
		return m, noop
	}
	if len(m.nav) == 0 {
		if m.mode != modeHome {
			m.showHome()
			return m, noop
		}
		m.status = "already at home"
		return m, noop
	}
	last := m.nav[len(m.nav)-1]
	m.nav = m.nav[:len(m.nav)-1]
	m.mode = last.mode
	if m.mode != modeStation {
		m.station = nil
	}
	m.refreshTitle()
	m.list.SetItems(last.items)
	if len(last.items) > 0 {
		m.list.Select(last.cursor)
	}
	m.status = last.status
	m.err = ""
	m.searching = false
	return m, noop
}
