package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestListEscQuitsLikeQ(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyRunes} {
		t.Run(key.String(), func(t *testing.T) {
			m := NewModel()
			m.mode = modeList
			m.filterText = "dev"

			msg := tea.KeyMsg{Type: key}
			if key == tea.KeyRunes {
				msg.Runes = []rune{'q'}
			}

			updated, cmd := m.updateList(msg)
			if cmd == nil {
				t.Fatal("expected quit command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("command returned %T, want tea.QuitMsg", cmd())
			}

			got := updated.(Model)
			if got.filterText != "dev" {
				t.Fatalf("filterText = %q, want unchanged filter", got.filterText)
			}
		})
	}
}
