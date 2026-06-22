package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) playBar(width int) string {
	state := "idle"
	title := "no track selected"
	artist := ""
	cover := "no cover"
	color := muted
	if m.playing != nil {
		state = "playing"
		color = crushMint
		if m.paused {
			state = "paused"
			color = crushGold
		}
		title = m.playing.Title
		if m.trackTitle != "" {
			title = m.trackTitle
		}
		artist = m.playing.Artist
		if m.playing.CoverID != "" {
			cover = "cover ready"
		}
	}
	left := lipgloss.NewStyle().Foreground(color).Bold(true).Render(strings.ToUpper(state))
	coverBadge := meterBadge(cover, crushPink)
	meta := title
	if artist != "" {
		meta = artist + " - " + title
	}
	maxMeta := max(8, width-lipgloss.Width(left)-lipgloss.Width(coverBadge)-8)
	line := fmt.Sprintf("%s  %s  %s", left, gradientText(ansi.Truncate(meta, maxMeta, "..."), trackStops), coverBadge)
	return m.styles.track.Width(width).Height(1).Render(line)
}
