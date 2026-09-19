package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mevanlc/mux/tmux"
)

func TestShortenPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{home + "/projects/foo", "~/projects/foo"},
		{home, "~"},
		{"/tmp/other", "/tmp/other"},
		{"", ""},
	}

	for _, tt := range tests {
		got := shortenPath(tt.input)
		if got != tt.want {
			t.Errorf("shortenPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShortenPathTruncatesLong(t *testing.T) {
	long := "/very/long/path/that/exceeds/thirty/five/characters/definitely"
	got := shortenPath(long)
	if len(got) > 35 {
		t.Errorf("shortenPath should truncate to 35 chars, got len=%d: %q", len(got), got)
	}
	if !strings.HasPrefix(got, "...") {
		t.Errorf("truncated path should start with '...', got %q", got)
	}
}

func TestAiLabelPlain(t *testing.T) {
	// Known commands should return non-empty
	for _, cmd := range []string{"claude", "codex", "aider", "gemini"} {
		info := aiLabelPlain(cmd)
		if info.styled == "" {
			t.Errorf("aiLabelPlain(%q) returned empty styled", cmd)
		}
		if info.text == "" {
			t.Errorf("aiLabelPlain(%q) returned empty text", cmd)
		}
		if info.extraWidth != 1 {
			t.Errorf("aiLabelPlain(%q) extraWidth = %d, want 1", cmd, info.extraWidth)
		}
	}
	// Unknown commands should return empty
	info := aiLabelPlain("bash")
	if info.styled != "" {
		t.Errorf("aiLabelPlain(%q) styled = %q, want empty", "bash", info.styled)
	}
}

func TestRenderPreviewNilSession(t *testing.T) {
	output := renderPreview(nil, "", 40, 10, nil)
	if !strings.Contains(output, "No session selected") {
		t.Error("nil session should show 'No session selected'")
	}
}

func TestRenderPreviewTabbedCapture(t *testing.T) {
	item := &listItem{kind: itemSession, session: &tmux.Session{Name: "shell"}}
	const capture = "\x1b[32m4.0K\x1b[0m\t./directory\n界\tcolumn\tnext\n12345678\tlast"

	for _, width := range []int{14, 33, 80, 133} {
		output := renderPreview(item, capture, width, 8, nil)
		if strings.Contains(output, "\t") {
			t.Errorf("width %d: preview contains terminal-dependent tabs", width)
		}
		lines := strings.Split(output, "\n")
		if len(lines) != 8 {
			t.Fatalf("width %d: got %d rows, want 8", width, len(lines))
		}
		for row, line := range lines {
			if got := ansi.StringWidth(line); got != width {
				t.Errorf("width %d, row %d: got %d cells", width, row, got)
			}
		}
		for i, expanded := range []string{
			"4.0K    ./directory",
			"界      column  next",
			"12345678        last",
		} {
			want := "│" + padOrTruncate(expanded, width-2) + "│"
			if got := ansi.Strip(lines[3+i]); got != want {
				t.Errorf("width %d, capture row %d: got %q, want %q", width, i, got, want)
			}
		}
		if !strings.Contains(lines[3], "\x1b[32m4.0K\x1b[0m") {
			t.Errorf("width %d: preview lost captured colors: %q", width, lines[3])
		}
	}
}
