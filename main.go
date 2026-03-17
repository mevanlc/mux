package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lunemis/mux/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If user selected a session to attach, exec into tmux
	if m, ok := result.(ui.Model); ok {
		if name := m.AttachName(); name != "" {
			if err := ui.AttachToSession(name); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to attach: %v\n", err)
				os.Exit(1)
			}
		}
	}
}
