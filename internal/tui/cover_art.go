package tui

import (
	"bytes"
	"context"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bprendie/subweazl/internal/subsonic"
)

func (m Model) loadCoverArt(id string) tea.Cmd {
	if id == "" {
		return nil
	}
	if img, ok := m.coverCache[id]; ok {
		return func() tea.Msg { return coverArtMsg{id: id, img: img} }
	}
	return func() tea.Msg {
		data, err := m.client.CoverArt(context.Background(), id, 500)
		if err != nil {
			return coverArtMsg{id: id, err: err}
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return coverArtMsg{id: id, err: err}
		}
		return coverArtMsg{id: id, img: img}
	}
}

func coverArtID(track subsonic.Track) string {
	if track.CoverID != "" {
		return track.CoverID
	}
	if track.AlbumID != "" {
		return track.AlbumID
	}
	return track.ID
}

func (m Model) coverPanel(width, height int) string {
	title := m.styles.help.Render("cover")
	if m.coverArt == nil {
		message := "no art"
		if m.coverErr != "" {
			message = m.coverErr
		} else if m.coverID != "" {
			message = "loading"
		}
		return m.styles.panel.Width(width).Height(height).Render(title + "\n\n" + m.styles.help.Render(message))
	}
	rendered := renderImageBlocks(m.coverArt, max(8, width-2), max(4, height-2))
	rendered = lipgloss.NewStyle().MaxWidth(width - 2).MaxHeight(height - 2).Render(rendered)
	return m.styles.panel.Width(width).Height(height).Render(rendered)
}

func renderImageBlocks(img image.Image, width, height int) string {
	if img == nil || width <= 0 || height <= 0 {
		return ""
	}
	bounds := img.Bounds()
	var b strings.Builder
	for y := 0; y < height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := 0; x < width; x++ {
			top := sampleColor(img, bounds, x, y*2, width, height*2)
			bottom := sampleColor(img, bounds, x, y*2+1, width, height*2)
			b.WriteString(lipgloss.NewStyle().Foreground(top).Background(bottom).Render("▀"))
		}
	}
	return b.String()
}

func sampleColor(img image.Image, bounds image.Rectangle, x, y, width, height int) lipgloss.Color {
	px := bounds.Min.X + x*bounds.Dx()/max(1, width)
	py := bounds.Min.Y + y*bounds.Dy()/max(1, height)
	return lipgloss.Color(hexColor(img.At(px, py)))
}

func hexColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return "#" + hexByte(uint8(r>>8)) + hexByte(uint8(g>>8)) + hexByte(uint8(b>>8))
}

func hexByte(v uint8) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>4], digits[v&0x0f]})
}
