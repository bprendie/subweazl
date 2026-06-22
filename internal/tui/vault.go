package tui

import (
	"errors"
	"strings"

	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	vaultStageCreate  = "create"
	vaultStageConfirm = "confirm"
	vaultStageUnlock  = "unlock"
)

func (m *Model) prepareVault() error {
	if m.vaultStore == nil {
		store, err := localstore.OpenDefault()
		if err != nil {
			m.mode = modeVault
			m.vaultStage = vaultStageUnlock
			m.vaultInput.Prompt = "vault password > "
			m.vaultInput.Focus()
			m.status = "private vault unavailable"
			return err
		}
		if err := store.Migrate(); err != nil {
			_ = store.Close()
			m.mode = modeVault
			m.vaultStage = vaultStageUnlock
			m.vaultInput.Prompt = "vault password > "
			m.vaultInput.Focus()
			m.status = "private vault unavailable"
			return err
		}
		m.vaultStore = store
	}
	hasVault, err := m.vaultStore.HasVault()
	if err != nil {
		m.mode = modeVault
		m.vaultStage = vaultStageUnlock
		m.vaultInput.Prompt = "vault password > "
		m.vaultInput.Focus()
		m.status = "private vault unavailable"
		return err
	}
	if m.vaultStore.Unlocked() {
		return nil
	}
	m.mode = modeVault
	m.vaultInput.SetValue("")
	m.vaultInput.EchoMode = textinput.EchoPassword
	m.vaultInput.Focus()
	m.err = ""
	if hasVault {
		m.vaultStage = vaultStageUnlock
		m.vaultInput.Prompt = "vault password > "
		m.status = "unlock private vault"
		return nil
	}
	m.vaultStage = vaultStageCreate
	m.vaultInput.Prompt = "new vault password > "
	m.status = "create private vault"
	return nil
}

func (m Model) handleVaultKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.stop()
		m.closeVaultStore()
		return m, tea.Quit
	case "enter":
		return m.submitVault()
	case "esc":
		if m.vaultStage == vaultStageConfirm {
			m.vaultPass = ""
			m.vaultStage = vaultStageCreate
			m.vaultInput.SetValue("")
			m.vaultInput.Prompt = "new vault password > "
			m.status = "vault creation cancelled"
			m.err = ""
		}
		return m, noop
	}
	var cmd tea.Cmd
	m.vaultInput, cmd = m.vaultInput.Update(msg)
	return m, cmd
}

func (m Model) submitVault() (Model, tea.Cmd) {
	password := strings.TrimSpace(m.vaultInput.Value())
	if password == "" {
		m.err = "vault password is required"
		return m, noop
	}
	if m.vaultStore == nil {
		if err := m.prepareVault(); err != nil {
			m.err = err.Error()
			return m, noop
		}
	}
	var err error
	switch m.vaultStage {
	case vaultStageCreate:
		m.vaultPass = password
		m.vaultStage = vaultStageConfirm
		m.vaultInput.SetValue("")
		m.vaultInput.Prompt = "confirm password > "
		m.status = "confirm private vault password"
		m.err = ""
		return m, noop
	case vaultStageConfirm:
		if password != m.vaultPass {
			m.vaultPass = ""
			m.vaultStage = vaultStageCreate
			m.vaultInput.SetValue("")
			m.vaultInput.Prompt = "new vault password > "
			m.status = "create private vault"
			m.err = "vault passwords did not match"
			return m, noop
		}
		err = m.vaultStore.CreateVault(m.vaultPass)
	case vaultStageUnlock:
		err = m.vaultStore.Unlock(password)
	default:
		err = errors.New("private vault action is unknown")
	}
	m.vaultPass = ""
	m.vaultInput.SetValue("")
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.mode = modeNewest
	m.refreshTitle()
	m.status = "private vault unlocked"
	m.err = ""
	m.restoreLastPlayed()
	if m.hasRestoredLastPlayed() {
		return m, m.loadCoverArt(m.coverID)
	}
	m.beginSearch("loading newest albums")
	return m, m.loadNewest()
}

func (m *Model) closeVaultStore() {
	if m.vaultStore != nil {
		_ = m.vaultStore.Close()
		m.vaultStore = nil
	}
}
