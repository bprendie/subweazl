package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bprendie/subweazl/internal/localindex"
	"github.com/bprendie/subweazl/internal/localstore"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) renderLocalLibrary() (Model, tea.Cmd) {
	m.clearNav()
	m.mode = modeLocal
	m.refreshTitle()
	m.err = ""
	m.searching = false
	m.resetInput()
	if err := m.ensureLocalStore(); err != nil {
		m.err = err.Error()
		m.list.SetItems(oneItem(item{kind: "empty", title: "Local vault unavailable", desc: "check database permissions"}))
		return m, noop
	}
	hasVault, err := m.localStore.HasVault()
	if err != nil {
		m.err = err.Error()
		m.list.SetItems(oneItem(item{kind: "empty", title: "Local vault unavailable", desc: "could not inspect vault"}))
		return m, noop
	}
	if !hasVault {
		m.list.SetItems(oneItem(item{
			kind:   "action",
			title:  "Create encrypted local vault",
			desc:   "enter a vault password before browsing local music",
			action: "local-vault",
		}))
		m.status = "local library needs an encrypted vault"
		return m, noop
	}
	if !m.localStore.Unlocked() {
		m.list.SetItems(oneItem(item{
			kind:   "action",
			title:  "Unlock encrypted local vault",
			desc:   "enter the vault password to decrypt folders and tracks",
			action: "local-vault",
		}))
		m.status = "local vault is locked"
		return m, noop
	}
	return m.renderUnlockedLocal()
}

func (m Model) renderUnlockedLocal() (Model, tea.Cmd) {
	items, status, err := m.localItems()
	if err != nil {
		m.err = err.Error()
		m.list.SetItems(oneItem(item{kind: "empty", title: "Local library unavailable", desc: "could not decrypt local records"}))
		return m, noop
	}
	m.list.SetItems(items)
	m.status = status
	return m, noop
}

func (m *Model) ensureLocalStore() error {
	if m.localStore != nil {
		return nil
	}
	store, err := localstore.OpenDefault()
	if err != nil {
		return err
	}
	if err := store.Migrate(); err != nil {
		_ = store.Close()
		return err
	}
	m.localStore = store
	return nil
}

func (m Model) startLocalVaultPrompt() (Model, tea.Cmd) {
	if err := m.ensureLocalStore(); err != nil {
		m.err = err.Error()
		return m, noop
	}
	hasVault, err := m.localStore.HasVault()
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	m.localVault = "unlock"
	prompt := "vault password > "
	status := "enter local vault password"
	if !hasVault {
		m.localVault = "create-password"
		prompt = "new vault password > "
		status = "enter new local vault password"
	}
	m.localPass = ""
	m.input.Prompt = prompt
	m.input.SetValue("")
	m.input.EchoMode = textinput.EchoPassword
	m.input.Focus()
	m.status = status
	m.err = ""
	return m, noop
}

func (m Model) submitLocalVault() (Model, tea.Cmd) {
	password := m.input.Value()
	if password == "" {
		return m, nil
	}
	if err := m.ensureLocalStore(); err != nil {
		m.err = err.Error()
		return m, noop
	}
	var err error
	switch m.localVault {
	case "create-password":
		m.localPass = password
		m.input.SetValue("")
		m.input.Prompt = "confirm password > "
		m.localVault = "create-confirm"
		m.status = "confirm local vault password"
		m.err = ""
		return m, noop
	case "create-confirm":
		if password != m.localPass {
			m.localPass = ""
			m.localVault = "create-password"
			m.input.SetValue("")
			m.input.Prompt = "new vault password > "
			m.status = "enter new local vault password"
			m.err = "vault passwords did not match"
			return m, noop
		}
		err = m.localStore.CreateVault(m.localPass)
	case "unlock":
		err = m.localStore.Unlock(password)
	default:
		err = errors.New("local vault action is unknown")
	}
	m.resetInput()
	if err != nil {
		m.err = err.Error()
		return m, noop
	}
	return m.renderUnlockedLocal()
}

func (m Model) localItems() ([]list.Item, string, error) {
	folders, err := m.localStore.FolderSummaries()
	if err != nil {
		return nil, "", err
	}
	tracks, err := m.localStore.TrackSummaries(200)
	if err != nil {
		return nil, "", err
	}
	m.ensureLocalRootsOpen(folders)
	items := make([]list.Item, 0, len(folders)+len(tracks)+len(m.cfg.LocalMusicFolders)+1)
	if len(m.cfg.LocalMusicFolders) > 0 {
		items = append(items, item{
			kind:   "action",
			title:  "Index configured folders",
			desc:   fmt.Sprintf("scan %s with ffprobe", m.localLabel()),
			action: "local-index",
		})
	}
	for _, folder := range localFolderItems(folders, tracks, m.cfg.LocalMusicFolders, m.localOpen) {
		items = append(items, folder)
	}
	for _, track := range tracks {
		if folderVisible(track.Path, m.localOpen) {
			items = append(items, item{
				kind: "local-song",
				local: localTrack{
					ID:      track.ID,
					Title:   track.Title,
					Artist:  track.Artist,
					Album:   track.Album,
					Path:    track.Path,
					Dir:     filepath.Dir(track.Path),
					Missing: track.Missing,
				},
			})
		}
	}
	if len(items) == 0 {
		items = append(items, item{kind: "empty", title: "No local folders", desc: "add folders in setup before local browsing"})
	}
	status := fmt.Sprintf("local vault unlocked: %d folders, %d tracks", len(folders), len(tracks))
	return items, status, nil
}

func (m Model) indexLocalFolders() (Model, tea.Cmd) {
	if m.localStore == nil || !m.localStore.Unlocked() {
		m.err = "unlock the local vault before indexing"
		return m, noop
	}
	if len(m.cfg.LocalMusicFolders) == 0 {
		m.err = "no local folders configured"
		return m, noop
	}
	spinnerCmd := m.beginSearch("indexing local folders")
	store := m.localStore
	folders := append([]string(nil), m.cfg.LocalMusicFolders...)
	indexCmd := func() tea.Msg {
		indexer := localindex.Indexer{Store: store, Prober: localindex.FFProbe{}}
		var msg localIndexedMsg
		for _, folder := range folders {
			result, err := indexer.IndexFolder(context.Background(), folder)
			if err != nil {
				return errMsg{err}
			}
			msg.folders++
			msg.indexed += result.Indexed
			msg.skipped += result.Skipped
		}
		return msg
	}
	return m, tea.Batch(spinnerCmd, indexCmd)
}

func (m *Model) resetInput() {
	m.input.SetValue("")
	m.input.Prompt = searchPrompt
	m.input.EchoMode = textinput.EchoNormal
	m.input.Blur()
	m.localVault = ""
	m.localPass = ""
}

func (m *Model) closeLocalStore() {
	if m.localStore != nil {
		_ = m.localStore.Close()
		m.localStore = nil
	}
}
