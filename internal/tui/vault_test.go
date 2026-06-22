package tui

import (
	"strings"
	"testing"

	"github.com/bprendie/subweazl/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestReadyConfigStartsInVaultMode(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	if m.mode != modeVault {
		t.Fatalf("mode = %v, want modeVault", m.mode)
	}
	if m.vaultStage != vaultStageCreate {
		t.Fatalf("vaultStage = %q, want create", m.vaultStage)
	}
	if m.vaultStore == nil {
		t.Fatal("vault store was not opened")
	}
}

func TestVaultCreateUnlocksAndLoadsNewest(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.vaultInput.SetValue("thisguy47")
	next, _ := m.submitVault()
	m = next
	if m.vaultStage != vaultStageConfirm {
		t.Fatalf("vaultStage = %q, want confirm", m.vaultStage)
	}
	m.vaultInput.SetValue("thisguy47")
	next, cmd := m.submitVault()
	m = next
	if m.mode != modeNewest {
		t.Fatalf("mode = %v, want modeNewest", m.mode)
	}
	if m.vaultStore == nil || !m.vaultStore.Unlocked() {
		t.Fatal("vault store is not unlocked")
	}
	if cmd == nil {
		t.Fatal("unlock should return a load command")
	}
}

func TestVaultPasswordMismatchReturnsToCreate(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.vaultInput.SetValue("one")
	next, _ := m.submitVault()
	m = next
	m.vaultInput.SetValue("two")
	next, _ = m.submitVault()
	m = next
	if m.vaultStage != vaultStageCreate {
		t.Fatalf("vaultStage = %q, want create", m.vaultStage)
	}
	if m.err == "" {
		t.Fatal("password mismatch did not report an error")
	}
}

func TestExistingVaultStartsInUnlockMode(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.vaultInput.SetValue("thisguy47")
	next, _ := m.submitVault()
	m = next
	m.vaultInput.SetValue("thisguy47")
	if next, _ = m.submitVault(); next.vaultStore == nil || !next.vaultStore.Unlocked() {
		t.Fatal("failed to create initial vault")
	}
	next.closeVaultStore()
	m = New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	if m.vaultStage != vaultStageUnlock {
		t.Fatalf("vaultStage = %q, want unlock", m.vaultStage)
	}
	m.vaultInput.SetValue("wrong")
	next, _ = m.submitVault()
	m = next
	if m.err == "" {
		t.Fatal("wrong password did not report an error")
	}
	m.vaultInput.SetValue("thisguy47")
	next, cmd := m.submitVault()
	if next.mode != modeNewest {
		t.Fatalf("mode = %v, want modeNewest", next.mode)
	}
	if cmd == nil {
		t.Fatal("unlock should return a load command")
	}
}

func TestVaultQuitClosesStore(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	if m.vaultStore == nil {
		t.Fatal("vault store was not opened")
	}
	next, cmd := m.handleVaultKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if next.vaultStore != nil {
		t.Fatal("vault store was not closed")
	}
	if cmd == nil {
		t.Fatal("quit should return a command")
	}
}

func TestVaultModeRendersVaultView(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("SUBWEAZL_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := New(config.Config{Server: "https://example.test", Username: "user", Password: "pass"})
	m.resize(100, 30)
	rendered := m.View()
	if !strings.Contains(rendered, "PRIVATE // PERSONAL // VAULTED") {
		t.Fatalf("vault view did not render vault panel:\n%s", rendered)
	}
}
