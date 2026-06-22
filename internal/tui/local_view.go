package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type indexedItem struct {
	index int
	item  item
}

func (m Model) localPane(width, height int) string {
	items := localIndexedItems(m.list.Items())
	if len(items.tracks) == 0 && len(items.folders) == 0 {
		return m.mainListPane(width, height)
	}
	leftWidth := clampInt(width/3, 24, 36)
	rightWidth := max(24, width-leftWidth-2)
	left := m.localBrowserPane(items, leftWidth, height)
	scope := m.selectedLocalFolder()
	right := m.localTrackPane(localScopedTracks(items.tracks, scope), rightWidth, height, scope)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) localBrowserPane(items localGroups, width, height int) string {
	var b strings.Builder
	b.WriteString(m.localVaultLine(width - 4))
	b.WriteByte('\n')
	for _, row := range items.actions {
		b.WriteString(m.localNavRow(row, width-4))
		b.WriteByte('\n')
	}
	if len(items.folders) > 0 {
		b.WriteByte('\n')
		b.WriteString(m.railHeader("FOLDERS"))
		b.WriteByte('\n')
	}
	for _, row := range trimIndexed(items.folders, max(2, height-8), m.list.Index()) {
		b.WriteString(m.localNavRow(row, width-4))
		b.WriteByte('\n')
	}
	return m.styles.panel.Width(width).Height(height).Render(b.String())
}

func (m Model) localTrackPane(tracks []indexedItem, width, height int, scope *localFolder) string {
	var b strings.Builder
	titleWidth := clampInt(width/2-5, 12, width-22)
	artistWidth := clampInt(width/4, 10, 24)
	albumWidth := max(8, width-titleWidth-artistWidth-13)
	b.WriteString(m.localTrackHeader(width-4, len(tracks), scope))
	b.WriteByte('\n')
	b.WriteString(m.styles.help.Render(
		padRight("#", 3) +
			padRight("TITLE", titleWidth) +
			padRight("ARTIST", artistWidth) +
			"ALBUM",
	))
	b.WriteByte('\n')
	rows := trimIndexed(tracks, max(1, height-5), m.list.Index())
	for _, row := range rows {
		line := localTrackRow(row.item.local, titleWidth, artistWidth, albumWidth)
		if row.index == m.list.Index() {
			b.WriteString(m.styles.selected.Render("> " + line))
		} else {
			b.WriteString("  " + m.styles.item.Render(line))
		}
		b.WriteByte('\n')
	}
	if len(tracks) == 0 {
		b.WriteString(m.styles.help.Render("  index configured folders to populate tracks"))
	}
	return m.styles.active.Width(width).Height(height).Render(b.String())
}

func (m Model) localVaultLine(width int) string {
	state := "LOCKED"
	color := crushGold
	if m.localStore != nil && m.localStore.Unlocked() {
		state = "UNLOCKED"
		color = crushMint
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(ansi.Truncate("LOCAL VAULT "+state, max(8, width), "..."))
}

func (m Model) localTrackHeader(width, count int, scope *localFolder) string {
	label := fmt.Sprintf("TRACKS %d", count)
	if scope != nil {
		name := filepath.Base(scope.Path)
		if name == "." || name == string(filepath.Separator) {
			name = scope.Path
		}
		label = fmt.Sprintf("TRACKS %d - %s", count, name)
	}
	return lipgloss.NewStyle().
		Foreground(crushGold).
		Bold(true).
		Render(ansi.Truncate(label, max(8, width), "..."))
}

func (m Model) selectedLocalFolder() *localFolder {
	it, ok := m.list.SelectedItem().(item)
	if !ok || it.kind != "local-folder" {
		return nil
	}
	folder := it.folder
	return &folder
}

func localScopedTracks(tracks []indexedItem, scope *localFolder) []indexedItem {
	if scope == nil {
		return tracks
	}
	out := make([]indexedItem, 0, len(tracks))
	for _, row := range tracks {
		dir := row.item.local.Dir
		if dir == "" {
			dir = filepath.Dir(row.item.local.Path)
		}
		if pathWithin(scope.Path, dir) {
			out = append(out, row)
		}
	}
	return out
}

func (m Model) localNavRow(row indexedItem, width int) string {
	label := row.item.Title()
	if row.item.kind == "local-folder" {
		label = filepath.Base(row.item.folder.Path)
		if label == "." || label == string(filepath.Separator) {
			label = row.item.folder.Path
		}
		twist := "▸ "
		if row.item.folder.Expanded {
			twist = "▾ "
		}
		label = strings.Repeat("  ", row.item.folder.Depth) + twist + label
	}
	prefix := "  "
	style := m.styles.item
	if row.index == m.list.Index() {
		prefix = "> "
		style = m.styles.selected
	}
	return prefix + style.Render(ansi.Truncate(label, max(8, width-2), "..."))
}

func localTrackRow(track localTrack, titleWidth, artistWidth, albumWidth int) string {
	title := track.Title
	if title == "" {
		name := filepath.Base(track.Path)
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return padRight("", 3) +
		padRight(ansi.Truncate(title, titleWidth-1, "..."), titleWidth) +
		padRight(ansi.Truncate(track.Artist, artistWidth-1, "..."), artistWidth) +
		ansi.Truncate(track.Album, albumWidth, "...")
}

type localGroups struct {
	actions []indexedItem
	folders []indexedItem
	tracks  []indexedItem
}

func localIndexedItems(rows []list.Item) localGroups {
	var groups localGroups
	for index, row := range rows {
		it, ok := row.(item)
		if !ok {
			continue
		}
		entry := indexedItem{index: index, item: it}
		switch it.kind {
		case "action":
			groups.actions = append(groups.actions, entry)
		case "local-folder":
			groups.folders = append(groups.folders, entry)
		case "local-song":
			groups.tracks = append(groups.tracks, entry)
		}
	}
	return groups
}

func trimIndexed(rows []indexedItem, limit int, selected int) []indexedItem {
	if limit <= 0 || len(rows) <= limit {
		return rows
	}
	pos := 0
	for i, row := range rows {
		if row.index == selected {
			pos = i
			break
		}
	}
	start := pos - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > len(rows) {
		start = len(rows) - limit
	}
	return rows[start : start+limit]
}

func padRight(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = ansi.Truncate(value, width, "...")
	for lipgloss.Width(value) < width {
		value += " "
	}
	return value
}
