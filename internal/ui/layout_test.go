package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/lunemis/mux/internal/tmux"
)

func TestLayoutDimensions(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "claude", Windows: 1, Created: time.Now().Add(-2 * time.Hour), Attached: true, Directory: "/Users/test/workspace/project1"},
		{Name: "dev-server", Windows: 2, Created: time.Now().Add(-24 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project2"},
		{Name: "deploy", Windows: 1, Created: time.Now().Add(-48 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project3"},
	}

	widths := []int{80, 120, 160, 200}
	heights := []int{20, 30, 40, 50}

	for _, w := range widths {
		for _, h := range heights {
			t.Run("", func(t *testing.T) {
				m := NewModel()
				m.width = w
				m.height = h
				m.sessions = sessions
				m.filtered = sessions
				m.cursor = 0

				output := m.viewMain()
				lines := strings.Split(output, "\n")

				t.Logf("w=%d h=%d => output lines=%d", w, h, len(lines))

				if len(lines) > h {
					t.Errorf("w=%d h=%d: output has %d lines, exceeds terminal height %d", w, h, len(lines), h)
					// Print first and last few lines for debugging
					for i, l := range lines {
						if i < 3 || i >= len(lines)-3 {
							t.Logf("  line %d (len=%d): %q", i, len(l), truncStr(l, 80))
						}
					}
				}
			})
		}
	}
}

func TestListPreviewSameHeight(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "claude", Windows: 1, Created: time.Now(), Attached: true, Directory: "/Users/test/project"},
		{Name: "dev", Windows: 1, Created: time.Now(), Attached: false, Directory: "/Users/test/dev"},
	}

	width := 120
	height := 30

	listWidth := width * 2 / 5
	previewWidth := width - listWidth
	contentHeight := height - 3

	listOut := renderSessionList(sessions, 0, "", listWidth, contentHeight)
	previewOut := renderPreview(&sessions[0], previewWidth, contentHeight)

	listLines := strings.Count(listOut, "\n") + 1
	previewLines := strings.Count(previewOut, "\n") + 1

	t.Logf("list lines=%d, preview lines=%d, contentHeight=%d", listLines, previewLines, contentHeight)

	if listLines != previewLines {
		t.Errorf("height mismatch: list=%d preview=%d", listLines, previewLines)
	}
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
