package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/mux/tmux"
)

func TestLayoutDimensions(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "claude", WindowCount: 1, Created: time.Now().Add(-2 * time.Hour), Attached: true, Directory: "/Users/test/workspace/project1"},
		{Name: "dev-server", WindowCount: 2, Created: time.Now().Add(-24 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project2"},
		{Name: "deploy", WindowCount: 1, Created: time.Now().Add(-48 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project3"},
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

func TestVerticalLayoutDimensions(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "claude", WindowCount: 1, Created: time.Now().Add(-2 * time.Hour), Attached: true, Directory: "/Users/test/workspace/project1"},
		{Name: "dev-server", WindowCount: 2, Created: time.Now().Add(-24 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project2"},
		{Name: "deploy", WindowCount: 1, Created: time.Now().Add(-48 * time.Hour), Attached: false, Directory: "/Users/test/workspace/project3"},
	}

	widths := []int{80, 120, 160, 200}
	heights := []int{20, 30, 40, 50}

	for _, w := range widths {
		for _, h := range heights {
			t.Run("", func(t *testing.T) {
				m := NewModel(WithLayoutMode(LayoutVertical))
				m.width = w
				m.height = h
				m.sessions = sessions
				m.filtered = sessions
				m.cursor = 0

				output := m.viewMain()
				lines := strings.Split(output, "\n")

				if len(lines) > h {
					t.Errorf("w=%d h=%d: vertical output has %d lines, exceeds terminal height %d", w, h, len(lines), h)
				}
			})
		}
	}
}

func TestAutoLayoutSwitchesWithAspectRatio(t *testing.T) {
	m := NewModel()

	m.width = 100
	m.height = 50
	if !m.isVerticalLayout() {
		t.Fatal("near-square terminal should use vertical layout")
	}

	m.width = 320
	m.height = 90
	if m.isVerticalLayout() {
		t.Fatal("16:9 terminal should use horizontal layout")
	}

	m.width = 25
	m.height = 9
	if m.isVerticalLayout() {
		t.Fatal("equidistant aspect ratio should keep horizontal default")
	}
}

func TestParseLayoutMode(t *testing.T) {
	tests := []struct {
		value     string
		wantMode  LayoutMode
		wantValue string
	}{
		{"a", LayoutAuto, "auto"},
		{"auto", LayoutAuto, "auto"},
		{"h", LayoutHorizontal, "horizontal"},
		{"horizontal", LayoutHorizontal, "horizontal"},
		{"v", LayoutVertical, "vertical"},
		{"vertical", LayoutVertical, "vertical"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			gotMode, gotValue, err := ParseLayoutMode(tt.value)
			if err != nil {
				t.Fatalf("ParseLayoutMode(%q): %v", tt.value, err)
			}
			if gotMode != tt.wantMode {
				t.Fatalf("mode = %v, want %v", gotMode, tt.wantMode)
			}
			if gotValue != tt.wantValue {
				t.Fatalf("value = %q, want %q", gotValue, tt.wantValue)
			}
		})
	}
}

func TestParseLayoutModeRejectsInvalidValue(t *testing.T) {
	if _, _, err := ParseLayoutMode("diagonal"); err == nil {
		t.Fatal("ParseLayoutMode should reject invalid values")
	}
}

func TestForcedVerticalOverridesAutoLayout(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutVertical))
	m.width = 320
	m.height = 90

	if !m.isVerticalLayout() {
		t.Fatal("forced vertical layout should override 16:9 terminal")
	}
}

func TestForcedHorizontalOverridesAutoLayout(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutHorizontal))
	m.width = 100
	m.height = 50

	if m.isVerticalLayout() {
		t.Fatal("forced horizontal layout should override near-square terminal")
	}
}

func TestSessionNameUsesAvailableRowWidth(t *testing.T) {
	name := "very-long-session-name-that-fits-here"
	row := formatSessionRow(tmux.Session{Name: name, Created: time.Now()}, false, false, 60)

	if !strings.Contains(row, name) {
		t.Fatalf("session name should not be capped when row has space: %q", row)
	}
	if got := ansi.StringWidth(row); got != 60 {
		t.Fatalf("row width = %d, want 60", got)
	}
}

func TestSessionNameTruncatesToNarrowRowWidth(t *testing.T) {
	name := "very-long-session-name-that-does-not-fit"
	row := formatSessionRow(tmux.Session{Name: name, Created: time.Now()}, false, false, 24)

	if strings.Contains(row, name) {
		t.Fatalf("session name should truncate in a narrow row: %q", row)
	}
	if !strings.Contains(row, "...") {
		t.Fatalf("truncated session name should include ellipsis: %q", row)
	}
	if got := ansi.StringWidth(row); got != 24 {
		t.Fatalf("row width = %d, want 24", got)
	}
}

func TestListPreviewSameHeight(t *testing.T) {
	sessions := []tmux.Session{
		{Name: "claude", WindowCount: 1, Created: time.Now(), Attached: true, Directory: "/Users/test/project"},
		{Name: "dev", WindowCount: 1, Created: time.Now(), Attached: false, Directory: "/Users/test/dev"},
	}

	width := 120
	height := 30

	listWidth := width * 2 / 5
	previewWidth := width - listWidth
	contentHeight := height - 3

	listOut := renderSessionList(sessions, 0, "", listWidth, contentHeight)
	item := &listItem{kind: itemSession, session: &sessions[0]}
	previewOut := renderPreview(item, "", previewWidth, contentHeight, nil)

	listLines := strings.Count(listOut, "\n") + 1
	previewLines := strings.Count(previewOut, "\n") + 1

	t.Logf("list lines=%d, preview lines=%d, contentHeight=%d", listLines, previewLines, contentHeight)

	if listLines != previewLines {
		t.Errorf("height mismatch: list=%d preview=%d", listLines, previewLines)
	}
}

func TestDrawBorderResetsBeforeRightBorder(t *testing.T) {
	out := drawBorder("\x1b[1mhello", 12, 1)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("drawBorder returned %d lines, want 3", len(lines))
	}

	contentIndex := strings.Index(lines[1], "\x1b[1mhello")
	resetIndex := strings.LastIndex(lines[1], ansiReset)
	if contentIndex < 0 || resetIndex < contentIndex {
		t.Fatalf("content line does not reset before right border: %q", lines[1])
	}
}

func TestSessionListScrolling(t *testing.T) {
	// Create more sessions than can fit in a small viewport
	sessions := make([]tmux.Session, 20)
	for i := range sessions {
		sessions[i] = tmux.Session{
			Name:        fmt.Sprintf("session-%02d", i),
			WindowCount: 1,
			Created:     time.Now(),
		}
	}

	width := 60
	height := 10 // innerHeight = 8, so only 8 sessions visible

	// Cursor at 0: first session should be visible
	out := renderSessionList(sessions, 0, "", width, height)
	if !strings.Contains(out, "session-00") {
		t.Error("cursor=0: expected session-00 to be visible")
	}

	// Cursor at 15: should scroll so session-15 is visible
	out = renderSessionList(sessions, 15, "", width, height)
	if !strings.Contains(out, "session-15") {
		t.Error("cursor=15: expected session-15 to be visible")
	}
	// session-00 should be scrolled out
	if strings.Contains(out, "session-00") {
		t.Error("cursor=15: expected session-00 to be scrolled out")
	}

	// Cursor at last session
	out = renderSessionList(sessions, 19, "", width, height)
	if !strings.Contains(out, "session-19") {
		t.Error("cursor=19: expected session-19 to be visible")
	}
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
