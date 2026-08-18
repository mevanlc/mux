package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mevanlc/mux/tmux"
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

func TestClickSessionRowSelectsSession(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutHorizontal))
	m.width = 100
	m.height = 30
	m.sessions = []tmux.Session{{Name: "first"}, {Name: "second"}}
	m.applyFilter()

	listX, listY, _, _ := m.listPanelBounds()
	msg := tea.MouseMsg{
		X:      listX + 5,
		Y:      listY + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	updated, cmd := m.updateList(msg)
	got := updated.(Model)

	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
	if cmd == nil {
		t.Fatal("expected selected session preview to refresh")
	}
}

func TestClickOutsideSessionRowDoesNothing(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutHorizontal))
	m.width = 100
	m.height = 30
	m.sessions = []tmux.Session{{Name: "a"}, {Name: "second"}}
	m.applyFilter()

	listX, listY, _, _ := m.listPanelBounds()
	for name, msg := range map[string]tea.MouseMsg{
		"border": {
			X:      listX,
			Y:      listY + 2,
			Button: tea.MouseButtonLeft,
			Action: tea.MouseActionPress,
		},
		"right button": {
			X:      listX + 5,
			Y:      listY + 2,
			Button: tea.MouseButtonRight,
			Action: tea.MouseActionPress,
		},
	} {
		t.Run(name, func(t *testing.T) {
			updated, cmd := m.updateList(msg)
			got := updated.(Model)
			if got.cursor != 0 {
				t.Fatalf("cursor = %d, want unchanged cursor 0", got.cursor)
			}
			if cmd != nil {
				t.Fatal("unexpected command for a click outside a session row")
			}
		})
	}
}

func TestClickSessionRowPaddingSelectsSession(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutHorizontal))
	m.width = 100
	m.height = 30
	m.sessions = []tmux.Session{{Name: "first"}, {Name: "second"}}
	m.applyFilter()

	listX, listY, listWidth, _ := m.listPanelBounds()
	msg := tea.MouseMsg{
		X:      listX + listWidth - 2,
		Y:      listY + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	updated, _ := m.updateList(msg)
	got := updated.(Model)
	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1 after clicking row padding", got.cursor)
	}
}

func TestClickSessionRowAccountsForFilterBar(t *testing.T) {
	m := NewModel(WithLayoutMode(LayoutVertical))
	m.width = 80
	m.height = 30
	m.filterText = "session"
	m.sessions = []tmux.Session{{Name: "session-one"}, {Name: "session-two"}}
	m.applyFilter()

	listX, listY, _, _ := m.listPanelBounds()
	if listY != 2 {
		t.Fatalf("list y = %d, want 2 with filter bar", listY)
	}
	msg := tea.MouseMsg{
		X:      listX + 5,
		Y:      listY + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}
	updated, _ := m.updateList(msg)
	got := updated.(Model)
	if got.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", got.cursor)
	}
}
