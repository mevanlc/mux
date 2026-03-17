package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/internal/tmux"
)

type mode int

const (
	modeList mode = iota
	modeCreate
	modeRename
	modeFilter
	modeConfirmKill
)

type Model struct {
	sessions    []tmux.Session
	filtered    []tmux.Session
	cursor      int
	mode        mode
	width       int
	height      int
	err         error
	createModel createModel
	renameModel renameModel
	filterInput textinput.Model
	filterText  string
	attachName  string // set when we want to attach after quitting
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type sessionsLoadedMsg struct {
	sessions []tmux.Session
	err      error
}

func loadSessions() tea.Msg {
	sessions, err := tmux.ListSessions()
	return sessionsLoadedMsg{sessions: sessions, err: err}
}

func NewModel() Model {
	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 50
	fi.Width = 30

	return Model{
		filterInput: fi,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSessions, tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(loadSessions, tick())

	case sessionsLoadedMsg:
		m.err = msg.err
		if msg.sessions != nil {
			m.sessions = msg.sessions
			m.applyFilter()
		}
		return m, nil

	case sessionCreatedMsg:
		m.mode = modeList
		return m, loadSessions

	case sessionRenamedMsg:
		m.mode = modeList
		return m, loadSessions
	}

	switch m.mode {
	case modeCreate:
		return m.updateCreate(msg)
	case modeRename:
		return m.updateRename(msg)
	case modeFilter:
		return m.updateFilter(msg)
	case modeConfirmKill:
		return m.updateConfirmKill(msg)
	default:
		return m.updateList(msg)
	}
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "g":
			m.cursor = 0
		case "G":
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
			}

		case "enter":
			if len(m.filtered) > 0 {
				m.attachName = m.filtered[m.cursor].Name
				return m, tea.Quit
			}

		case "n":
			m.mode = modeCreate
			m.createModel = newCreateModel()
			return m, m.createModel.nameInput.Focus()

		case "x":
			if len(m.filtered) > 0 {
				m.mode = modeConfirmKill
			}

		case "r":
			if len(m.filtered) > 0 {
				m.mode = modeRename
				m.renameModel = newRenameModel(m.filtered[m.cursor].Name)
				return m, m.renameModel.input.Focus()
			}

		case "/":
			m.mode = modeFilter
			m.filterInput.SetValue(m.filterText)
			return m, m.filterInput.Focus()

		case "esc":
			if m.filterText != "" {
				m.filterText = ""
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.mode = modeList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.createModel, cmd = m.createModel.Update(msg)
	return m, cmd
}

func (m Model) updateRename(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			m.mode = modeList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.renameModel, cmd = m.renameModel.Update(msg)
	return m, cmd
}

func (m Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.mode = modeList
			m.filterText = ""
			m.applyFilter()
			return m, nil
		case "enter":
			m.mode = modeList
			m.filterText = m.filterInput.Value()
			m.applyFilter()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	// Live filter as you type
	m.filterText = m.filterInput.Value()
	m.applyFilter()
	return m, cmd
}

func (m Model) updateConfirmKill(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			if m.cursor < len(m.filtered) {
				name := m.filtered[m.cursor].Name
				if err := tmux.KillSession(name); err != nil {
					m.err = err
				}
			}
			m.mode = modeList
			return m, loadSessions
		default:
			m.mode = modeList
		}
	}
	return m, nil
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = m.sessions
	} else {
		lower := strings.ToLower(m.filterText)
		m.filtered = nil
		for _, s := range m.sessions {
			if strings.Contains(strings.ToLower(s.Name), lower) ||
				strings.Contains(strings.ToLower(s.Directory), lower) {
				m.filtered = append(m.filtered, s)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.mode {
	case modeCreate:
		return m.viewWithOverlay(m.createModel.View())
	case modeRename:
		return m.viewWithOverlay(m.renameModel.View())
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	// Title
	count := fmt.Sprintf("(%d)", len(m.filtered))
	title := titleStyle.Render("⚡ tmux sessions " + count)

	// Help bar
	help := renderHelp()

	// Filter / confirm bar
	var extraBar string
	if m.mode == modeFilter {
		extraBar = inputLabelStyle.Render("/ ") + m.filterInput.View()
	} else if m.mode == modeConfirmKill && m.cursor < len(m.filtered) {
		extraBar = errorStyle.Render(
			fmt.Sprintf("Kill \"%s\"? (y/N)", m.filtered[m.cursor].Name))
	} else if m.filterText != "" {
		extraBar = helpStyle.Render(fmt.Sprintf("filter: %s (esc clear)", m.filterText))
	}

	// Chrome: title(1+margin1) + help(1) + extraBar(0 or 1)
	chrome := 3
	if extraBar != "" {
		chrome++
	}

	// Panel height = total height for both borders + content
	panelHeight := m.height - chrome
	if panelHeight < 5 {
		panelHeight = 5
	}

	// Layout: list on left (40%), preview on right (60%)
	listWidth := m.width * 2 / 5
	previewWidth := m.width - listWidth

	// Render both panels (each returns exactly panelHeight lines)
	list := renderSessionList(m.filtered, m.cursor, m.filterText, listWidth, panelHeight)

	var currentSession *tmux.Session
	if m.cursor < len(m.filtered) {
		currentSession = &m.filtered[m.cursor]
	}
	preview := renderPreview(currentSession, previewWidth, panelHeight)

	// Join line-by-line for exact alignment
	content := joinHorizontalFixed(list, preview)

	// Assemble
	var b strings.Builder
	b.WriteString(title)
	b.WriteByte('\n')
	if extraBar != "" {
		b.WriteString(extraBar)
		b.WriteByte('\n')
	}
	b.WriteString(content)
	b.WriteByte('\n')
	b.WriteString(help)

	return b.String()
}

func (m Model) viewWithOverlay(overlay string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Render(overlay)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

func renderHelp() string {
	keys := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"enter", "attach"},
		{"n", "new"},
		{"x", "kill"},
		{"r", "rename"},
		{"/", "filter"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			helpKeyStyle.Render(k.key)+" "+helpStyle.Render(k.desc))
	}
	return strings.Join(parts, helpStyle.Render("  •  "))
}

// AttachName returns the session to attach to (if any) after TUI exits
func (m Model) AttachName() string {
	return m.attachName
}

// AttachToSession switches to the target session.
// If already inside tmux, uses switch-client. Otherwise, uses attach-session.
func AttachToSession(name string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	if os.Getenv("TMUX") != "" {
		// Inside tmux: switch-client doesn't need exec, just run it
		return exec.Command(tmuxPath, "switch-client", "-t", name).Run()
	}
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}
