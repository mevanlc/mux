package ui

import (
	"path/filepath"
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
	if _, err := settingsPath(); err != nil {
		t.Fatalf("settings path: %v", err)
	}
	if got, want := mustSettingsPath(t), filepath.Join(dir, "mux", configFileName); got != want {
		t.Fatalf("settings path = %q, want %q", got, want)
	}
}

func TestVerticalSplitResizePersistsIndependently(t *testing.T) {
	withTempSettingsDir(t)

	m := NewModel(WithVerticalLayout())
	m.height = 30
	updated, _ := m.updateList(tea.KeyMsg{Type: tea.KeyShiftDown})
	m = updated.(Model)

	if got := splitPaneSize(27, m.verticalSplit); got != 12 {
		t.Fatalf("resized vertical split = %d, want 12", got)
	}
	if got := splitPaneSize(100, m.horizontalSplit); got != 40 {
		t.Fatalf("horizontal split = %d, want default 40", got)
	}

	restored := NewModel(WithVerticalLayout())
	if got := splitPaneSize(27, restored.verticalSplit); got != 12 {
		t.Fatalf("restored vertical split = %d, want 12", got)
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
