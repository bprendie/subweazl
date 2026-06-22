package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m Model) View() string {
	contentWidth := max(20, m.width-4)
	if m.mode == modeSetup {
		var b strings.Builder
		if contentWidth >= maxLineWidth(logo) {
			b.WriteString(renderLogo(logo, contentWidth))
		} else {
			b.WriteString(m.styles.header.Render("Subweazl"))
		}
		b.WriteString("\n\n")
		b.WriteString(m.setupView(contentWidth))
		return m.styles.frame.Render(b.String())
	}
	return m.styles.frame.Render(m.appShell(contentWidth))
}

func (m Model) appShell(width int) string {
	var b strings.Builder
	header := m.appHeader(width)
	footer := m.footer(width)
	innerHeight := max(6, m.height-2)
	bodyRenderedHeight := innerHeight - lipgloss.Height(header) - lipgloss.Height(footer) - 2
	bodyRenderedHeight = max(4, bodyRenderedHeight)
	bodyHeight := max(2, bodyRenderedHeight-2)
	body := ""
	if width < 64 {
		body = m.mainListPane(width, bodyHeight)
	} else {
		sidebarWidth := clampInt(width/4, 20, 30)
		mainWidth := max(24, width-sidebarWidth-2)
		sidebar := m.sidebar(sidebarWidth, bodyHeight)
		main := m.mainListPane(mainWidth, bodyHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	}
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(footer)
	if m.err != "" {
		b.WriteString("\n" + m.styles.error.Render(m.err))
	}
	if !m.cfg.Ready() {
		b.WriteString("\n" + m.styles.error.Render("set SUBWEAZL_USER and SUBWEAZL_PASSWORD or edit config.json"))
	}
	return b.String()
}

func (m Model) mainListPane(width, height int) string {
	content := m.list.View()
	style := m.styles.active
	return style.Width(width).Height(height).Render(content)
}

func (m Model) footer(width int) string {
	if width < 64 {
		return lipgloss.JoinVertical(lipgloss.Left, m.playBar(width), m.statusLine(), m.styles.help.Render(m.helpMenu(width)))
	}
	artWidth := 18
	visualWidth := max(18, width/4)
	playerWidth := max(20, width-artWidth-visualWidth-4)
	art := m.coverPanel(artWidth, 7)
	viz := m.styles.panel.Width(visualWidth).Height(5).Render(m.visualizer.View(m.styles))
	player := m.playBar(playerWidth)
	status := m.statusLine()
	help := m.styles.help.Render(m.helpMenu(width))
	top := lipgloss.JoinHorizontal(lipgloss.Top, art, viz, player)
	return lipgloss.JoinVertical(lipgloss.Left, top, status, help)
}

func (m Model) statusLine() string {
	if m.searching {
		return m.styles.status.Render(m.spinner.View() + " " + ansi.Wordwrap(m.status, max(20, m.width-6), " /_-"))
	}
	if m.isPlaying() {
		state := "now: "
		if m.paused {
			state = "paused: "
		}
		meter := " " + meterBadge("SYNTH", crushGold)
		if m.energy.Live {
			meter = " " + meterBadge("LIVE", crushMint)
		}
		return m.styles.status.Render(ansi.Wordwrap(state+m.playingLabel()+meter, max(20, m.width-4), " /_-"))
	}
	return m.styles.status.Render(ansi.Wordwrap(m.status, max(20, m.width-4), " /_-"))
}

func meterBadge(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Border(lipgloss.NormalBorder(), false, true, false, true).
		BorderForeground(crushPurple).
		Padding(0, 1).
		Render(label)
}

func tick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
