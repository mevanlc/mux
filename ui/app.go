// Package ui implements the Bubble Tea TUI for browsing and managing tmux sessions.
package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lunemis/mux/tmux"
)

const (
	// Layout
	defaultSplitRatio = 0.4
	minPanelHeight    = 5
	minSplitPaneSize  = 5

	// Timing
	refreshInterval = 500 * time.Millisecond

	// Display limits
	maxSessionNameDisplay = 18
	maxPathDisplay        = 35
	filterCharLimit       = 50
	filterInputWidth      = 30
)

type mode int

const (
	modeList mode = iota
	modeCreate
	modeRename
	modeFilter
	modeConfirmKill
)

// LayoutMode controls whether mux chooses a layout automatically or forces one.
type LayoutMode int

const (
	LayoutAuto LayoutMode = iota
	LayoutHorizontal
	LayoutVertical
)

func ParseLayoutMode(value string) (LayoutMode, string, error) {
	switch strings.ToLower(value) {
	case "a", "auto":
		return LayoutAuto, "auto", nil
	case "h", "horizontal":
		return LayoutHorizontal, "horizontal", nil
	case "v", "vertical":
		return LayoutVertical, "vertical", nil
	default:
		return LayoutAuto, "", fmt.Errorf("invalid layout %q (want auto/a, horizontal/h, or vertical/v)", value)
	}
}

// Model is the top-level Bubble Tea model for the session manager TUI.
type Model struct {
	sessions        []tmux.Session
	filtered        []tmux.Session
	items           []listItem // flattened tree of (sessions, windows, panes)
	tree            treeState
	cursor          int
	mode            mode
	width           int
	height          int
	layoutMode      LayoutMode
	horizontalSplit float64
	verticalSplit   float64
	err             error
	createModel     createModel
	renameModel     renameModel
	filterMod       filterModel
	confirmKillMod  confirmKillModel
	filterText      string
	attachTarget    previewKey       // set when we want to attach after quitting (zero value = no attach)
	focusSession    string           // session name to focus cursor on after next load
	previewContent  string           // cached capture-pane output
	previewKey      previewKey       // (session, window, pane) the cache belongs to
	tokenUsage      *tmux.TokenUsage // cached token usage for current AI session
	tokenSession    string           // session name the token cache belongs to
	splitResizeDrag *splitResizeDrag // active mouse drag of the panel divider
}

type splitResizeDrag struct {
	layout     splitLayout
	origin     int
	total      int
	grabOffset int
}

// ModelOption configures the TUI model.
type ModelOption func(*Model)

// WithLayoutMode controls whether the TUI uses automatic, horizontal, or
// vertical layout.
func WithLayoutMode(layoutMode LayoutMode) ModelOption {
	return func(m *Model) {
		m.layoutMode = layoutMode
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
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

type previewLoadedMsg struct {
	key     previewKey
	content string
}

type tokenUsageLoadedMsg struct {
	sessionName string
	usage       *tmux.TokenUsage
}

type windowsLoadedMsg struct {
	sessionName string
	windows     []tmux.Window
}

type panesLoadedMsg struct {
	sessionName string
	windowIndex int
	panes       []tmux.Pane
}

func loadWindows(sessionName string) tea.Cmd {
	return func() tea.Msg {
		windows, _ := tmux.ListWindows(sessionName)
		return windowsLoadedMsg{sessionName: sessionName, windows: windows}
	}
}

func loadPanes(sessionName string, windowIndex int) tea.Cmd {
	return func() tea.Msg {
		panes, _ := tmux.ListPanes(sessionName, windowIndex)
		return panesLoadedMsg{sessionName: sessionName, windowIndex: windowIndex, panes: panes}
	}
}

func refreshPreview(key previewKey) tea.Cmd {
	return func() tea.Msg {
		content, err := tmux.CapturePaneTarget(key.target())
		if err != nil {
			content = "Error: " + err.Error()
		}
		return previewLoadedMsg{key: key, content: content}
	}
}

func loadTokenUsage(sessionName string, panePID int) tea.Cmd {
	return func() tea.Msg {
		sessionID, cwd, err := tmux.FindClaudeSession(panePID)
		if err != nil {
			return tokenUsageLoadedMsg{sessionName: sessionName}
		}
		usage, _ := tmux.LoadTokenUsage(sessionID, cwd)
		return tokenUsageLoadedMsg{sessionName: sessionName, usage: usage}
	}
}

// NewModel returns a new Model with default settings.
func NewModel(options ...ModelOption) Model {
	settings := loadSettings()
	m := Model{
		tree:            newTreeState(),
		horizontalSplit: settings.splitRatio(layoutHorizontal),
		verticalSplit:   settings.splitRatio(layoutVertical),
	}
	for _, option := range options {
		option(&m)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSessions, tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.splitResizeDrag = nil
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{loadSessions, tick()}
		if it := m.currentItem(); it != nil {
			cmds = append(cmds, refreshPreview(previewKeyForItem(*it)))
			if tmux.IsAICommand(it.session.ActiveCommand) {
				cmds = append(cmds, loadTokenUsage(it.session.Name, it.session.PanePID))
			}
		}
		// Refresh windows/panes for expanded subtrees
		for name := range m.tree.expandedSession {
			cmds = append(cmds, loadWindows(name))
		}
		for sessionName, windows := range m.tree.expandedWindow {
			for windowIdx := range windows {
				cmds = append(cmds, loadPanes(sessionName, windowIdx))
			}
		}
		return m, tea.Batch(cmds...)

	case sessionsLoadedMsg:
		m.err = msg.err
		if msg.sessions != nil {
			m.sessions = msg.sessions
			m.tree.pruneCaches(m.sessions)
			m.applyFilter()
			if m.focusSession != "" {
				for i, it := range m.items {
					if it.kind == itemSession && it.session.Name == m.focusSession {
						m.cursor = i
						break
					}
				}
				m.focusSession = ""
			}
		}
		return m, nil

	case windowsLoadedMsg:
		m.tree.windowsCache[msg.sessionName] = msg.windows
		m.rebuildItems()
		return m, nil

	case panesLoadedMsg:
		m.tree.panesCache[paneCacheKey{session: msg.sessionName, window: msg.windowIndex}] = msg.panes
		m.rebuildItems()
		return m, nil

	case previewLoadedMsg:
		m.previewKey = msg.key
		m.previewContent = msg.content
		return m, nil

	case tokenUsageLoadedMsg:
		m.tokenSession = msg.sessionName
		m.tokenUsage = msg.usage
		return m, nil

	case sessionCreatedMsg:
		m.mode = modeList
		m.focusSession = msg.name
		return m, loadSessions

	case sessionRenamedMsg:
		m.mode = modeList
		return m, loadSessions

	case filterAppliedMsg:
		m.mode = modeList
		m.filterText = msg.text
		m.applyFilter()
		return m, nil

	case sessionKilledMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.mode = modeList
		if msg.name != "" {
			return m, loadSessions
		}
		return m, nil
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
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case "shift+left", "shift+up":
			return m.resizeSplit(-1)
		case "shift+right", "shift+down":
			return m.resizeSplit(1)

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.refreshCurrentPreview()
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				return m, m.refreshCurrentPreview()
			}
		case "g":
			m.cursor = 0
			return m, m.refreshCurrentPreview()
		case "G":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				return m, m.refreshCurrentPreview()
			}

		case "tab", "right", "l":
			return m.expandCurrent()

		case "shift+tab", "left", "h":
			return m.collapseCurrent()

		case "enter", "a":
			if it := m.currentItem(); it != nil {
				m.attachTarget = previewKeyForItem(*it)
				return m, tea.Quit
			}

		case "n":
			m.mode = modeCreate
			m.createModel = newCreateModel()
			return m, m.createModel.nameInput.Focus()

		case "x":
			if it := m.currentItem(); it != nil && it.kind == itemSession {
				m.mode = modeConfirmKill
				m.confirmKillMod = newConfirmKillModel(it.session.Name)
			}

		case "r":
			if it := m.currentItem(); it != nil && it.kind == itemSession {
				m.mode = modeRename
				m.renameModel = newRenameModel(it.session.Name)
				return m, m.renameModel.input.Focus()
			}

		case "/":
			m.mode = modeFilter
			m.filterMod = newFilterModel(m.filterText)
			return m, nil

		}
	case tea.MouseMsg:
		mouse := tea.MouseEvent(msg)
		switch {
		case mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionPress:
			if m.splitDividerHit(mouse.X, mouse.Y) {
				m.beginSplitResize(mouse.X, mouse.Y)
				return m, nil
			}

			listX, listY, listWidth, listHeight := m.listPanelBounds()
			idx := hitTestSessionRow(
				m.items,
				m.cursor,
				mouse.X-listX,
				mouse.Y-listY,
				listWidth,
				listHeight,
			)
			if idx >= 0 {
				m.cursor = idx
				return m, m.refreshCurrentPreview()
			}
		case mouse.Button == tea.MouseButtonLeft && mouse.Action == tea.MouseActionMotion:
			m.resizeSplitFromMouse(mouse.X, mouse.Y)
			return m, nil
		case mouse.Action == tea.MouseActionRelease && m.splitResizeDrag != nil:
			m.resizeSplitFromMouse(mouse.X, mouse.Y)
			layout := m.splitResizeDrag.layout
			ratio := m.horizontalSplit
			if layout == layoutVertical {
				ratio = m.verticalSplit
			}
			m.splitResizeDrag = nil
			_ = saveSplitRatio(layout, ratio)
			return m, nil
		default:
			m.splitResizeDrag = nil
		}
	}
	return m, nil
}

func (m Model) listPanelBounds() (x, y, width, height int) {
	y = 1 // title
	if m.hasExtraBar() {
		y++
	}

	panelHeight := m.panelHeight()
	if m.isVerticalLayout() {
		return 0, y, m.width, splitPaneSize(panelHeight, m.verticalSplit)
	}
	return 0, y, splitPaneSize(m.width, m.horizontalSplit), panelHeight
}

func (m Model) hasExtraBar() bool {
	return m.mode == modeFilter || m.mode == modeConfirmKill || m.filterText != ""
}

func (m Model) splitDividerHit(x, y int) bool {
	listX, listY, listWidth, listHeight := m.listPanelBounds()
	if m.isVerticalLayout() {
		inPanelColumn := x >= listX && x < listX+listWidth
		return inPanelColumn && (y == listY+listHeight-1 || y == listY+listHeight)
	}

	inPanelRow := y >= listY && y < listY+listHeight
	return inPanelRow && (x == listX+listWidth-1 || x == listX+listWidth)
}

func (m *Model) beginSplitResize(x, y int) {
	listX, listY, listWidth, listHeight := m.listPanelBounds()
	if m.isVerticalLayout() {
		m.splitResizeDrag = &splitResizeDrag{
			layout:     layoutVertical,
			origin:     listY,
			total:      m.panelHeight(),
			grabOffset: listHeight - (y - listY),
		}
		return
	}

	m.splitResizeDrag = &splitResizeDrag{
		layout:     layoutHorizontal,
		origin:     listX,
		total:      m.width,
		grabOffset: listWidth - (x - listX),
	}
}

func (m *Model) resizeSplitFromMouse(x, y int) {
	if m.splitResizeDrag == nil {
		return
	}

	drag := *m.splitResizeDrag
	coordinate := x
	if drag.layout == layoutVertical {
		coordinate = y
	}
	ratio := splitRatioForSize(drag.total, coordinate-drag.origin+drag.grabOffset)
	if drag.layout == layoutVertical {
		m.verticalSplit = ratio
	} else {
		m.horizontalSplit = ratio
	}
}

// expandCurrent expands the row under the cursor and dispatches the loader.
// On a pane (leaf) it does nothing.
func (m Model) expandCurrent() (tea.Model, tea.Cmd) {
	it := m.currentItem()
	if it == nil || !it.canExpand() {
		return m, nil
	}
	switch it.kind {
	case itemSession:
		if m.tree.isSessionExpanded(it.session.Name) {
			return m, nil
		}
		m.tree.setSessionExpanded(it.session.Name, true)
		m.rebuildItems()
		return m, loadWindows(it.session.Name)
	case itemWindow:
		if m.tree.isWindowExpanded(it.session.Name, it.window.Index) {
			return m, nil
		}
		m.tree.setWindowExpanded(it.session.Name, it.window.Index, true)
		m.rebuildItems()
		return m, loadPanes(it.session.Name, it.window.Index)
	}
	return m, nil
}

// collapseCurrent collapses the row under the cursor. On a child row whose own
// kind cannot collapse further, it walks up to the parent and collapses that.
func (m Model) collapseCurrent() (tea.Model, tea.Cmd) {
	it := m.currentItem()
	if it == nil {
		return m, nil
	}
	switch it.kind {
	case itemSession:
		if !m.tree.isSessionExpanded(it.session.Name) {
			return m, nil
		}
		m.tree.setSessionExpanded(it.session.Name, false)
	case itemWindow:
		if m.tree.isWindowExpanded(it.session.Name, it.window.Index) {
			m.tree.setWindowExpanded(it.session.Name, it.window.Index, false)
		} else {
			// Already-collapsed window: jump up to the parent session
			m.cursor = m.findItemIndex(itemSession, it.session.Name, 0, 0)
			m.tree.setSessionExpanded(it.session.Name, false)
		}
	case itemPane:
		// Collapse the parent window and move cursor up to it
		m.cursor = m.findItemIndex(itemWindow, it.session.Name, it.window.Index, 0)
		m.tree.setWindowExpanded(it.session.Name, it.window.Index, false)
	}
	m.rebuildItems()
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
	return m, m.refreshCurrentPreview()
}

// refreshCurrentPreview returns a tea.Cmd to capture the pane targeted by the
// current cursor position. Returns nil when there is no current item.
func (m *Model) refreshCurrentPreview() tea.Cmd {
	if it := m.currentItem(); it != nil {
		return refreshPreview(previewKeyForItem(*it))
	}
	return nil
}

func (m Model) resizeSplit(delta int) (tea.Model, tea.Cmd) {
	total := m.splitTotal()
	if total <= 1 {
		return m, nil
	}

	ratio := m.horizontalSplit
	if m.isVerticalLayout() {
		ratio = m.verticalSplit
	}

	size := splitPaneSize(total, ratio) + delta
	ratio = splitRatioForSize(total, size)
	if m.isVerticalLayout() {
		m.verticalSplit = ratio
	} else {
		m.horizontalSplit = ratio
	}

	_ = saveSplitRatio(m.splitLayout(), ratio)
	return m, nil
}

func (m Model) splitTotal() int {
	if m.isVerticalLayout() {
		if m.height == 0 {
			return 0
		}
		return m.panelHeight()
	}
	return m.width
}

func (m Model) splitLayout() splitLayout {
	if m.isVerticalLayout() {
		return layoutVertical
	}
	return layoutHorizontal
}

func (m Model) isVerticalLayout() bool {
	switch m.layoutMode {
	case LayoutHorizontal:
		return false
	case LayoutVertical:
		return true
	default:
		return autoVerticalLayout(m.width, m.height)
	}
}

// autoVerticalLayout compares the terminal aspect ratio with 1:1 and 16:9,
// accounting for terminal cells being roughly twice as tall as they are wide.
// Equal distance keeps the historical horizontal default.
func autoVerticalLayout(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	return 9*width < 25*height
}

func (m Model) panelHeight() int {
	chrome := 3
	if m.hasExtraBar() {
		chrome++
	}

	height := m.height - chrome
	if height < minPanelHeight {
		return minPanelHeight
	}
	return height
}

func splitPaneSize(total int, ratio float64) int {
	if total <= 1 {
		return total
	}
	if ratio <= 0 || ratio >= 1 {
		ratio = defaultSplitRatio
	}

	size := int(float64(total)*ratio + 0.5)
	return clampSplitPaneSize(total, size)
}

func splitRatioForSize(total, size int) float64 {
	if total <= 1 {
		return defaultSplitRatio
	}
	return float64(clampSplitPaneSize(total, size)) / float64(total)
}

func clampSplitPaneSize(total, size int) int {
	minSize := minSplitPaneSize
	if total < minSize*2 {
		minSize = 1
	}

	maxSize := total - minSize
	if maxSize < minSize {
		return max(1, total/2)
	}
	return min(max(size, minSize), maxSize)
}

// findItemIndex returns the index of the matching listItem, or -1 if not found.
func (m *Model) findItemIndex(kind itemKind, sessionName string, windowIdx, paneIdx int) int {
	for i, it := range m.items {
		if it.kind != kind || it.session.Name != sessionName {
			continue
		}
		switch kind {
		case itemSession:
			return i
		case itemWindow:
			if it.window.Index == windowIdx {
				return i
			}
		case itemPane:
			if it.window.Index == windowIdx && it.pane.Index == paneIdx {
				return i
			}
		}
	}
	return -1
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
	var cmd tea.Cmd
	m.filterMod, cmd = m.filterMod.Update(msg)
	// Live filter as you type
	m.filterText = m.filterMod.LiveText()
	m.applyFilter()
	return m, cmd
}

func (m Model) updateConfirmKill(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirmKillMod, cmd = m.confirmKillMod.Update(msg)
	return m, cmd
}

func (m *Model) currentItem() *listItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return &m.items[m.cursor]
	}
	return nil
}

// currentSession returns the parent session of the current row (the row itself
// for session rows). Returns nil if no row is selected.
func (m *Model) currentSession() *tmux.Session {
	if it := m.currentItem(); it != nil {
		return it.session
	}
	return nil
}

func (m *Model) currentSessionName() string {
	if s := m.currentSession(); s != nil {
		return s.Name
	}
	return ""
}

// rebuildItems recomputes the flattened tree view from the filtered session
// list and current expansion state. Call after sessions, filter, or expansion
// state changes.
func (m *Model) rebuildItems() {
	m.items = flatten(m.filtered, &m.tree)
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
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
	m.rebuildItems()
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
	// Title — count sessions only, not windows/panes
	count := fmt.Sprintf("(%d)", len(m.filtered))
	title := titleStyle.Render("⚡ tmux sessions " + count)

	// Help bar
	help := renderHelp()

	// Filter / confirm bar
	var extraBar string
	if m.mode == modeFilter {
		extraBar = m.filterMod.View()
	} else if m.mode == modeConfirmKill {
		extraBar = m.confirmKillMod.View()
	} else if m.filterText != "" {
		extraBar = helpStyle.Render(fmt.Sprintf("filter: %s", m.filterText))
	}

	// Panel height = total height for both borders + content
	panelHeight := m.panelHeight()

	currentItem := m.currentItem()
	currentSession := m.currentSession()
	cachedContent := ""
	if currentItem != nil && m.previewKey == previewKeyForItem(*currentItem) {
		cachedContent = m.previewContent
	}
	var tokenUsage *tmux.TokenUsage
	if currentSession != nil && m.tokenSession == currentSession.Name {
		tokenUsage = m.tokenUsage
	}

	var content string
	if m.isVerticalLayout() {
		listHeight := splitPaneSize(panelHeight, m.verticalSplit)
		previewHeight := panelHeight - listHeight

		list := renderListView(m.items, m.cursor, m.filterText, &m.tree, m.width, listHeight)
		preview := renderPreview(currentItem, cachedContent, m.width, previewHeight, tokenUsage)
		content = list + "\n" + preview
	} else {
		// Layout: list on left, preview on right
		listWidth := splitPaneSize(m.width, m.horizontalSplit)
		previewWidth := m.width - listWidth

		// Render both panels (each returns exactly panelHeight lines)
		list := renderListView(m.items, m.cursor, m.filterText, &m.tree, listWidth, panelHeight)
		preview := renderPreview(currentItem, cachedContent, previewWidth, panelHeight, tokenUsage)

		// Join line-by-line for exact alignment
		content = joinHorizontalFixed(list, preview)
	}

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
		{"←↑↓→", "nav"},
		{"a", "attach"},
		{"n", "new"},
		{"x", "kill"},
		{"r", "rename"},
		{"/", "filter"},
		{"q/esc", "quit"},
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			helpKeyStyle.Render(k.key)+" "+helpStyle.Render(k.desc))
	}
	return strings.Join(parts, helpStyle.Render(" • "))
}

// AttachName returns the session name to attach to (if any) after the TUI
// exits. Returns empty when no attach was requested.
func (m Model) AttachName() string {
	return m.attachTarget.session
}

// AttachWindowIndex returns the window index selected for attachment, or -1
// if the user selected a session row.
func (m Model) AttachWindowIndex() int {
	return m.attachTarget.window
}

// AttachPaneIndex returns the pane index selected for attachment, or -1 if
// the user did not drill down to a pane row.
func (m Model) AttachPaneIndex() int {
	return m.attachTarget.pane
}

// AttachToSession switches to the target session, optionally focusing a
// specific window and pane first. Pass windowIdx == -1 to keep the active
// window; pass paneIdx == -1 to keep the active pane within that window.
//
// If already inside tmux, uses switch-client. Otherwise, uses attach-session.
func AttachToSession(name string, windowIdx, paneIdx int) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}

	// Focus the requested window/pane *before* attaching, since attach-session
	// replaces our process and we can't run anything afterwards.
	if windowIdx >= 0 {
		windowTarget := fmt.Sprintf("%s:%d", name, windowIdx)
		if err := exec.Command(tmuxPath, "select-window", "-t", windowTarget).Run(); err != nil {
			return fmt.Errorf("select-window %s: %w", windowTarget, err)
		}
		if paneIdx >= 0 {
			paneTarget := fmt.Sprintf("%s.%d", windowTarget, paneIdx)
			if err := exec.Command(tmuxPath, "select-pane", "-t", paneTarget).Run(); err != nil {
				return fmt.Errorf("select-pane %s: %w", paneTarget, err)
			}
		}
	}

	if os.Getenv("TMUX") != "" {
		return exec.Command(tmuxPath, "switch-client", "-t", name).Run()
	}
	return syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
}
