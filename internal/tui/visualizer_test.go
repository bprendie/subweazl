package tui

import (
	"strings"
	"testing"
)

func TestVisualizerRendersSingleLine(t *testing.T) {
	v := NewVisualizer(0.016)
	v.Step(true)
	got := v.View(newStyles())
	if strings.Contains(got, "\n") {
		t.Fatalf("visualizer rendered multiple lines: %q", got)
	}
	if strings.Contains(got, " ") {
		t.Fatalf("visualizer rendered spacing that can wrap: %q", got)
	}
}
