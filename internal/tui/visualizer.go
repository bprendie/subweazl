package tui

import (
	"math"
	"strings"

	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

type Visualizer struct {
	spring     harmonica.Spring
	bars       []float64
	velocities []float64
	tick       int
}

func NewVisualizer(delta float64) Visualizer {
	return Visualizer{
		spring:     harmonica.NewSpring(delta, 9.0, 0.35),
		bars:       make([]float64, 24),
		velocities: make([]float64, 24),
	}
}

func (v *Visualizer) Step(playing bool) {
	v.tick++
	for i := range v.bars {
		base := 2.0
		if playing {
			base = 3 + 13*v.energyAt(i)
		}
		target := base
		if playing {
			target += 5 * (0.5 + 0.5*math.Sin(float64(v.tick)*0.12+float64(i)*0.9))
		}
		v.bars[i], v.velocities[i] = v.spring.Update(v.bars[i], v.velocities[i], target)
	}
}

func (v Visualizer) energyAt(i int) float64 {
	return 0.5 + 0.5*math.Sin(float64(v.tick+i)*0.35)
}

func (v Visualizer) View(styles styles) string {
	var b strings.Builder
	blocks := []rune("▁▂▃▄▅▆▇█")
	for i, value := range v.bars {
		b.WriteString(lipgloss.NewStyle().Foreground(v.color(i)).Render(string(blocks[v.index(value)])))
	}
	return b.String()
}

func (v Visualizer) index(value float64) int {
	idx := int(math.Round(value / 2))
	if idx < 0 {
		return 0
	}
	if idx > 7 {
		return 7
	}
	return idx
}

func (v Visualizer) color(i int) lipgloss.Color {
	if i%3 == 0 {
		return accent
	}
	if i%3 == 1 {
		return success
	}
	return secondary
}
