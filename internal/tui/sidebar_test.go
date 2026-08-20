package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTabFocusesSidebarAtActiveEntry(t *testing.T) {
	m := Model{width: 100, mode: modeQueue}
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if next.focus != focusSidebar {
		t.Fatal("tab should focus the sidebar")
	}
	if got := sidebarNavEntries[next.sidebarIndex].key; got != "4" {
		t.Fatalf("sidebar cursor = %q, want active queue entry", got)
	}
}

func TestSidebarNavigationAndActivation(t *testing.T) {
	m := Model{width: 100, mode: modeHome, focus: focusSidebar, sidebarIndex: 4}
	m.list = list.New(nil, delegate{styles: newStyles()}, 80, 20)
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	if got := sidebarNavEntries[next.sidebarIndex].key; got != "5" {
		t.Fatalf("sidebar cursor = %q, want private playlists", got)
	}
	next, _ = next.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modePrivatePlaylists {
		t.Fatalf("mode = %v, want private playlists", next.mode)
	}
	if next.focus != focusMain {
		t.Fatal("activating a sidebar entry should return focus to the list")
	}
}

func TestTabReturnsFocusToMainPane(t *testing.T) {
	m := Model{width: 100, focus: focusSidebar}
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	if next.focus != focusMain {
		t.Fatal("tab should return focus to the main pane")
	}
}

func TestNarrowLayoutCannotRetainSidebarFocus(t *testing.T) {
	m := Model{focus: focusSidebar}
	m.list = list.New(nil, delegate{styles: newStyles()}, 80, 20)
	m.resize(60, 30)
	if m.focus != focusMain {
		t.Fatal("hidden sidebar should not retain focus")
	}
}
