package main

import (
	"fmt"
	"os"

	"github.com/bprendie/subweazl/internal/config"
	"github.com/bprendie/subweazl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "subweazl: %v\n", err)
		os.Exit(1)
	}
}
