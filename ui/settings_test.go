package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func withTempSettingsDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	old := userConfigDir
	userConfigDir = func() (string, error) {
		return dir, nil
	}
	t.Cleanup(func() {
		userConfigDir = old
	})
	return dir
}

func TestHorizontalSplitResizePersists(t *testing.T) {
	dir := withTempSettingsDir(t)

	m := NewModel()
	m.width = 100
	updated, _ := m.updateList(tea.KeyMsg{Type: tea.KeyShiftRight})
	m = updated.(Model)

	if got := splitPaneSize(100, m.horizontalSplit); got != 41 {
		t.Fatalf("resized horizontal split = %d, want 41", got)
	}

	restored := NewModel()
	if got := splitPaneSize(100, restored.horizontalSplit); got != 41 {
		t.Fatalf("restored horizontal split = %d, want 41", got)
	}
	if got := splitPaneSize(100, restored.verticalSplit); got != 40 {
		t.Fatalf("restored vertical split = %d, want default 40", got)
	}
	if _, err := settingsPath(); err != nil {
		t.Fatalf("settings path: %v", err)
	}
	if got, want := mustSettingsPath(t), filepath.Join(dir, "mux", configFileName); got != want {
		t.Fatalf("settings path = %q, want %q", got, want)
	}
	data, err := os.ReadFile(mustSettingsPath(t))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if !strings.Contains(string(data), `"split_sizes"`) {
		t.Fatalf("settings should persist split sizes per layout: %s", data)
	}
	if strings.Contains(string(data), `"horizontal_split"`) || strings.Contains(string(data), `"vertical_split"`) {
		t.Fatalf("settings should not write legacy split fields: %s", data)
	}
}

func TestVerticalSplitResizePersistsIndependently(t *testing.T) {
	withTempSettingsDir(t)

	m := NewModel(WithLayoutMode(LayoutVertical))
	m.height = 30
	updated, _ := m.updateList(tea.KeyMsg{Type: tea.KeyShiftDown})
	m = updated.(Model)

	if got := splitPaneSize(27, m.verticalSplit); got != 12 {
		t.Fatalf("resized vertical split = %d, want 12", got)
	}
	if got := splitPaneSize(100, m.horizontalSplit); got != 40 {
		t.Fatalf("horizontal split = %d, want default 40", got)
	}

	restored := NewModel(WithLayoutMode(LayoutVertical))
	if got := splitPaneSize(27, restored.verticalSplit); got != 12 {
		t.Fatalf("restored vertical split = %d, want 12", got)
	}
}

func TestSplitResizePreservesOtherLayoutPersistedSize(t *testing.T) {
	withTempSettingsDir(t)

	m := NewModel()
	if err := saveSplitRatio(layoutVertical, 0.7); err != nil {
		t.Fatalf("save vertical split: %v", err)
	}

	m.width = 100
	updated, _ := m.updateList(tea.KeyMsg{Type: tea.KeyShiftRight})
	m = updated.(Model)

	restored := NewModel()
	if got := splitPaneSize(100, restored.horizontalSplit); got != 41 {
		t.Fatalf("restored horizontal split = %d, want 41", got)
	}
	if got := splitPaneSize(100, restored.verticalSplit); got != 70 {
		t.Fatalf("restored vertical split = %d, want 70", got)
	}
}

func TestHorizontalSplitDividerDragResizesAndPersists(t *testing.T) {
	for name, grabX := range map[string]int{
		"list border":    39,
		"preview border": 40,
	} {
		t.Run(name, func(t *testing.T) {
			withTempSettingsDir(t)

			m := NewModel(WithLayoutMode(LayoutHorizontal))
			m.width = 100
			m.height = 30
			m.cursor = 1

			updated, _ := m.updateList(tea.MouseMsg{
				X: grabX, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
			})
			m = updated.(Model)
			if m.splitResizeDrag == nil {
				t.Fatal("divider press did not start a resize drag")
			}

			dragX := 60
			if grabX == 39 {
				dragX = 59
			}
			updated, _ = m.updateList(tea.MouseMsg{
				X: dragX, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
			})
			m = updated.(Model)
			if got := splitPaneSize(100, m.horizontalSplit); got != 60 {
				t.Fatalf("dragged horizontal split = %d, want 60", got)
			}
			if m.cursor != 1 {
				t.Fatalf("cursor = %d, want unchanged cursor 1", m.cursor)
			}
			if _, err := os.Stat(mustSettingsPath(t)); !os.IsNotExist(err) {
				t.Fatalf("settings should not be saved before mouse release: %v", err)
			}

			updated, _ = m.updateList(tea.MouseMsg{
				X: dragX, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
			})
			m = updated.(Model)
			if m.splitResizeDrag != nil {
				t.Fatal("resize drag should end on mouse release")
			}

			restored := NewModel(WithLayoutMode(LayoutHorizontal))
			if got := splitPaneSize(100, restored.horizontalSplit); got != 60 {
				t.Fatalf("restored horizontal split = %d, want 60", got)
			}
		})
	}
}

func TestVerticalSplitDividerDragResizesAndPersists(t *testing.T) {
	withTempSettingsDir(t)

	m := NewModel(WithLayoutMode(LayoutVertical))
	m.width = 80
	m.height = 30
	_, listY, _, listHeight := m.listPanelBounds()
	grabY := listY + listHeight - 1

	updated, _ := m.updateList(tea.MouseMsg{
		X: 20, Y: grabY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m = updated.(Model)
	updated, _ = m.updateList(tea.MouseMsg{
		X: 20, Y: 15, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	m = updated.(Model)
	if got := splitPaneSize(m.panelHeight(), m.verticalSplit); got != 15 {
		t.Fatalf("dragged vertical split = %d, want 15", got)
	}

	updated, _ = m.updateList(tea.MouseMsg{
		X: 20, Y: 15, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease,
	})
	m = updated.(Model)
	if m.splitResizeDrag != nil {
		t.Fatal("resize drag should end on mouse release")
	}

	restored := NewModel(WithLayoutMode(LayoutVertical))
	if got := splitPaneSize(27, restored.verticalSplit); got != 15 {
		t.Fatalf("restored vertical split = %d, want 15", got)
	}
}

func TestSplitDividerDragClampsToMinimumPanelSize(t *testing.T) {
	withTempSettingsDir(t)

	m := NewModel(WithLayoutMode(LayoutHorizontal))
	m.width = 100
	m.height = 30

	updated, _ := m.updateList(tea.MouseMsg{
		X: 40, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress,
	})
	m = updated.(Model)
	updated, _ = m.updateList(tea.MouseMsg{
		X: 99, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	m = updated.(Model)
	if got := splitPaneSize(100, m.horizontalSplit); got != 95 {
		t.Fatalf("maximum horizontal split = %d, want 95", got)
	}

	updated, _ = m.updateList(tea.MouseMsg{
		X: 0, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion,
	})
	m = updated.(Model)
	if got := splitPaneSize(100, m.horizontalSplit); got != 5 {
		t.Fatalf("minimum horizontal split = %d, want 5", got)
	}
}

func mustSettingsPath(t *testing.T) string {
	t.Helper()
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
