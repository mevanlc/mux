package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const ansiReset = "\x1b[0m"

// padOrTruncate ensures a string is exactly `width` visible characters
func padOrTruncate(s string, width int) string {
	w := ansi.StringWidth(s)
	if w > width {
		return ansi.Truncate(s, width, "")
	}
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func truncateWithEllipsis(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= width {
		return s
	}
	if width <= 3 {
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width-3, "") + "..."
}

// fixedBox takes rendered content and forces it to exactly width x height visible area.
// It splits by newlines, truncates/pads each line to width, and truncates/pads to height lines.
func fixedBox(content string, width, height int) string {
	lines := strings.Split(content, "\n")

	result := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			result[i] = padOrTruncate(lines[i], width)
		} else {
			result[i] = strings.Repeat(" ", width)
		}
	}
	return strings.Join(result, "\n")
}

// joinHorizontalFixed joins two blocks of text side-by-side, line by line
func joinHorizontalFixed(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	result := make([]string, maxLen)
	for i := 0; i < maxLen; i++ {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		result[i] = l + r
	}
	return strings.Join(result, "\n")
}

// drawBorder wraps content lines with a rounded border
func drawBorder(content string, width, height int) string {
	innerWidth := width - 2
	lines := strings.Split(content, "\n")
	borderStyle := lipgloss.NewStyle().Foreground(colorBorder)
	leftBorder := borderStyle.Render("│")
	rightBorder := borderStyle.Render("│")

	// Build bordered output
	result := make([]string, 0, height+2)

	// Top border
	result = append(result, borderStyle.Render("╭"+strings.Repeat("─", innerWidth)+"╮"))

	// Content lines (pad/truncate to exactly height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		line = padOrTruncate(line, innerWidth)
		result = append(result, leftBorder+line+ansiReset+rightBorder)
	}

	// Bottom border
	result = append(result, borderStyle.Render("╰"+strings.Repeat("─", innerWidth)+"╯"))

	return strings.Join(result, "\n")
}
