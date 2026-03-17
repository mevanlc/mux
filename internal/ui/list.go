package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/internal/tmux"
)

func renderSessionList(sessions []tmux.Session, cursor int, filter string, width, height int) string {
	innerWidth := width - 2 // border chars
	innerHeight := height - 2

	if len(sessions) == 0 {
		msg := "No tmux sessions found"
		if filter != "" {
			msg = fmt.Sprintf("No match: \"%s\"", filter)
		}
		// Center the message
		lines := make([]string, innerHeight)
		mid := innerHeight / 2
		for i := range lines {
			if i == mid {
				lines[i] = padOrTruncate(centerText(msg, innerWidth), innerWidth)
			} else {
				lines[i] = strings.Repeat(" ", innerWidth)
			}
		}
		content := strings.Join(lines, "\n")
		return drawBorder(content, width, innerHeight)
	}

	lines := make([]string, innerHeight)
	for i := 0; i < innerHeight; i++ {
		if i < len(sessions) {
			lines[i] = formatSessionRow(sessions[i], i == cursor, innerWidth)
		} else {
			lines[i] = strings.Repeat(" ", innerWidth)
		}
	}

	content := strings.Join(lines, "\n")
	return drawBorder(content, width, innerHeight)
}

func formatSessionRow(s tmux.Session, selected bool, width int) string {
	var status string
	if s.Attached {
		status = "●"
	} else {
		status = "○"
	}

	name := s.Name
	if len(name) > 18 {
		name = name[:15] + "..."
	}

	ago := timeAgo(s.Created)

	raw := fmt.Sprintf(" %s %-18s %s %dw", status, name, ago, s.Windows)

	if selected {
		styled := lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCursor).
			Background(colorSelected).
			Render(padOrTruncate(raw, width))
		return styled
	}

	styled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render(padOrTruncate(raw, width))
	return styled
}

func centerText(s string, width int) string {
	pad := (width - len(s)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + s
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%3ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%3dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%3dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%3dd", int(d.Hours()/24))
	}
}
