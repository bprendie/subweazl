package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type delegate struct {
	styles styles
}

func (d delegate) Height() int  { return 2 }
func (d delegate) Spacing() int { return 0 }
func (d delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d delegate) Render(w io.Writer, m list.Model, index int, row list.Item) {
	it, ok := row.(item)
	if !ok || m.Width() <= 0 {
		return
	}
	width := max(10, m.Width()-4)
	prefix := d.kindBadge(it.kind)
	title := ansi.Truncate(it.Title(), max(8, width-lipgloss.Width(prefix)-2), "...")
	desc := firstLines(ansi.Wordwrap(it.Description(), width, " /_-"), 1)
	if index == m.Index() {
		title = d.styles.selected.Render("> ") + prefix + gradientText(title, trackStops)
		fmt.Fprintf(w, "%s\n%s", title, lipgloss.NewStyle().Foreground(crushPurple).Render("  "+desc))
		return
	}
	fmt.Fprintf(w, "%s\n%s", d.styles.item.Render(prefix+title), d.styles.help.Render("  "+desc))
}

func (d delegate) kindBadge(kind string) string {
	if kind == "" {
		kind = "song"
	}
	label := strings.ToUpper(kind[:1])
	return lipgloss.NewStyle().
		Foreground(crushGold).
		Border(lipgloss.NormalBorder(), false, true, false, true).
		BorderForeground(crushPurple).
		Padding(0, 1).
		Render(label) + " "
}

func firstLines(s string, limit int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= limit {
		return s
	}
	return strings.Join(lines[:limit], "\n")
}
