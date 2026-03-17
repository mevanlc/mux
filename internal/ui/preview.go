package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/internal/tmux"
)

func renderPreview(session *tmux.Session, width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2

	if session == nil {
		lines := make([]string, innerHeight)
		mid := innerHeight / 2
		msg := "No session selected"
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

	// Header
	header := fmt.Sprintf("[ %s ]  %s", session.Name, shortenPath(session.Directory))
	headerStyled := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(
		padOrTruncate(header, innerWidth))
	separator := lipgloss.NewStyle().Foreground(colorBorder).Render(
		strings.Repeat("─", innerWidth))

	// Capture pane content
	captured, err := tmux.CapturePane(session.Name)
	if err != nil {
		captured = "Error: " + err.Error()
	}

	// Available lines for content (minus header + separator)
	contentLines := innerHeight - 2
	if contentLines < 1 {
		contentLines = 1
	}

	capLines := strings.Split(captured, "\n")
	// Keep last N lines (most recent output)
	if len(capLines) > contentLines {
		capLines = capLines[len(capLines)-contentLines:]
	}

	// Build all lines: header, separator, then content
	allLines := make([]string, innerHeight)
	allLines[0] = headerStyled
	allLines[1] = separator
	for i := 0; i < contentLines; i++ {
		if i < len(capLines) {
			allLines[i+2] = padOrTruncate(capLines[i], innerWidth)
		} else {
			allLines[i+2] = strings.Repeat(" ", innerWidth)
		}
	}

	content := strings.Join(allLines, "\n")
	return drawBorder(content, width, innerHeight)
}

func shortenPath(path string) string {
	if idx := strings.Index(path, "/Users/"); idx >= 0 {
		parts := strings.SplitN(path[idx:], "/", 4)
		if len(parts) >= 4 {
			path = "~/" + parts[3]
		} else if len(parts) == 3 {
			path = "~"
		}
	}
	if len(path) > 35 {
		path = "..." + path[len(path)-32:]
	}
	return path
}
