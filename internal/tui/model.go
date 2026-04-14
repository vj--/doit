package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/vj/doit/internal/config"
	"github.com/vj/doit/internal/store"
	"github.com/vj/doit/internal/tasks"
)

// reloadInterval is how often we poll the on-disk mtime while idle.
// Small enough to feel responsive, large enough to stay negligible.
const reloadInterval = 3 * time.Second

type mode int

const (
	modeBoard mode = iota
	modePromptCreate
	modeEdit
	modeConfirmDelete
	modeHelp
	modeFilter
)

type Model struct {
	ctx      context.Context
	cfg      config.Config
	store    *store.Store
	board    tasks.Board
	keymap   keymap

	mode    mode
	width   int
	height  int

	focusCol, focusTask int
	filter              string
	showHidden          bool

	titleInput  textinput.Model
	bodyInput   textarea.Model
	labelsInput textinput.Model
	filterInput textinput.Model
	editingNew  bool

	status    string
	statusErr bool

	saving   bool
	spinner  spinner.Model
	help     help.Model
	mdWidth  int
	renderer *glamour.TermRenderer
}

// detailPaneWidth returns the width reserved for the right-hand task detail
// pane. It shrinks to 0 when the terminal is too narrow to fit it alongside
// at least one column of reasonable width.
func (m *Model) detailPaneWidth() int {
	if m.width < 80 {
		return 0
	}
	w := m.width / 3
	if w < 32 {
		w = 32
	}
	if w > 60 {
		w = 60
	}
	return w
}

func (m *Model) rebuildRenderer() {
	w := m.detailPaneWidth() - 4
	if w < 20 {
		w = 20
	}
	if w == m.mdWidth && m.renderer != nil {
		return
	}
	// Tie glamour's palette to lipgloss's resolved background so --theme /
	// DOIT_THEME / config theme actually take effect inside the detail pane.
	// WithAutoStyle runs glamour's own background probe which breaks inside
	// tmux and some SSH sessions, yielding unreadable washed-out text.
	style := "dark"
	if !lipgloss.HasDarkBackground() {
		style = "light"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(w),
	)
	if err != nil {
		return
	}
	m.renderer = r
	m.mdWidth = w
}

// NewModel constructs the initial TUI model. If the board file does not exist
// the model starts in modePromptCreate and lets the user confirm creation.
func NewModel(ctx context.Context, cfg config.Config, s *store.Store) (*Model, error) {
	m := &Model{
		ctx:    ctx,
		cfg:    cfg,
		store:  s,
		keymap: newKeymap(),
	}
	m.titleInput = textinput.New()
	m.titleInput.Placeholder = "Task title"
	m.titleInput.CharLimit = 200
	m.bodyInput = textarea.New()
	m.bodyInput.Placeholder = "Details (optional)"
	m.bodyInput.SetHeight(5)
	m.labelsInput = textinput.New()
	m.labelsInput.Placeholder = "comma-separated, e.g. bug, urgent"
	m.labelsInput.CharLimit = 200
	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "Filter (substring)"
	m.filterInput.CharLimit = 200

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colPurple)
	m.spinner = sp

	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(colPink).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(colMuted)
	h.Styles.FullKey = h.Styles.ShortKey
	h.Styles.FullDesc = h.Styles.ShortDesc
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(colFaint)
	h.Styles.FullSeparator = h.Styles.ShortSeparator
	m.help = h

	if !s.Exists() {
		m.mode = modePromptCreate
		return m, nil
	}
	b, err := s.Load()
	if err != nil {
		return nil, fmt.Errorf("load board: %w", err)
	}
	m.board = b
	return m, nil
}

func (m *Model) Init() tea.Cmd { return tea.Batch(tickReloadCmd(), m.spinner.Tick) }

// -- Reload ticker --

type reloadTickMsg struct{}

func tickReloadCmd() tea.Cmd {
	return tea.Tick(reloadInterval, func(time.Time) tea.Msg { return reloadTickMsg{} })
}

// -- Messages --

type savedMsg struct {
	err       error
	committed bool
	msg       string
}

type boardLoadedMsg struct {
	board tasks.Board
	err   error
}

// -- Commands --

func (m *Model) saveCmd(commitMsg string) tea.Cmd {
	// Deep-copy the board so subsequent in-memory mutations cannot race with
	// the in-flight save (Board.Columns and Column.Tasks are slices).
	board := cloneBoard(m.board)
	return func() tea.Msg {
		committed, err := m.store.Save(m.ctx, board, commitMsg)
		return savedMsg{err: err, committed: committed, msg: commitMsg}
	}
}

func cloneBoard(b tasks.Board) tasks.Board {
	out := tasks.Board{Title: b.Title, Columns: make([]tasks.Column, len(b.Columns))}
	for i, c := range b.Columns {
		nc := tasks.Column{ID: c.ID, Title: c.Title, Tasks: make([]tasks.Task, len(c.Tasks))}
		for j, t := range c.Tasks {
			nt := t
			if len(t.Labels) > 0 {
				nt.Labels = append([]string(nil), t.Labels...)
			}
			nc.Tasks[j] = nt
		}
		out.Columns[i] = nc
	}
	return out
}

func (m *Model) reloadIfExternallyModified() tea.Cmd {
	return func() tea.Msg {
		changed, err := m.store.ExternallyModified()
		if err != nil || !changed {
			return nil
		}
		b, err := m.store.Reload()
		return boardLoadedMsg{board: b, err: err}
	}
}

// -- Update --

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.bodyInput.SetWidth(msg.Width - 10)
		m.filterInput.Width = msg.Width - 10
		m.labelsInput.Width = msg.Width - 10
		m.help.Width = msg.Width
		m.rebuildRenderer()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case reloadTickMsg:
		// Idle poll. Skip while a save is in flight to avoid clobbering an
		// optimistically-updated in-memory board with a stale read.
		if m.saving || m.mode == modeEdit || m.mode == modeConfirmDelete {
			return m, tickReloadCmd()
		}
		return m, tea.Batch(m.reloadIfExternallyModified(), tickReloadCmd())

	case savedMsg:
		m.saving = false
		if msg.err != nil {
			m.statusErr = true
			m.status = "save failed: " + msg.err.Error()
			return m, nil
		}
		m.statusErr = false
		if msg.committed {
			m.status = "✓ " + msg.msg
		} else {
			m.status = "saved (no commit)"
		}
		return m, nil

	case boardLoadedMsg:
		if msg.err != nil {
			m.statusErr = true
			m.status = "reload failed: " + msg.err.Error()
			return m, nil
		}
		m.board = msg.board
		m.clampFocus()
		m.status = "reloaded from disk"
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m *Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modePromptCreate:
		return m.updatePromptCreate(msg)
	case modeEdit:
		return m.updateEdit(msg)
	case modeConfirmDelete:
		return m.updateConfirmDelete(msg)
	case modeHelp:
		m.mode = modeBoard
		return m, nil
	case modeFilter:
		return m.updateFilter(msg)
	}
	return m.updateBoard(msg)
}

func (m *Model) updatePromptCreate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if m.saving {
			return m, nil
		}
		m.board = tasks.DefaultBoard()
		m.mode = modeBoard
		m.saving = true
		return m, m.saveCmd("doit: create board")
	case "n", "N", "esc", "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// isMutatingKey reports whether a board-mode key would trigger a save.
// Used to gate user input while a previous save is still in-flight, matching
// the "single-slot in-flight queue" rule from CLAUDE.md.
func (m *Model) isMutatingKey(msg tea.KeyMsg) bool {
	k := m.keymap
	return key.Matches(msg, k.MoveLeft) || key.Matches(msg, k.MoveRight) ||
		key.Matches(msg, k.MoveUp) || key.Matches(msg, k.MoveDown) ||
		key.Matches(msg, k.New) || key.Matches(msg, k.Edit) ||
		key.Matches(msg, k.Delete)
}

func (m *Model) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := m.keymap

	if m.saving && m.isMutatingKey(msg) {
		m.statusErr = true
		m.status = "save in progress; try again in a moment"
		return m, nil
	}

	switch {
	case key.Matches(msg, k.Quit):
		return m, tea.Quit
	case key.Matches(msg, k.Help):
		m.mode = modeHelp
		return m, nil
	case key.Matches(msg, k.Filter):
		m.mode = modeFilter
		m.filterInput.SetValue(m.filter)
		m.filterInput.Focus()
		return m, textinput.Blink
	case key.Matches(msg, k.ToggleHidden):
		m.showHidden = !m.showHidden
		if m.showHidden {
			m.status = "showing hidden done tasks"
		} else {
			m.status = "hiding old done tasks"
		}
		m.clampFocusToVisible()
		return m, nil
	case key.Matches(msg, k.Left):
		if len(m.board.Columns) > 0 {
			m.focusCol = (m.focusCol - 1 + len(m.board.Columns)) % len(m.board.Columns)
			m.focusTask = 0
			m.clampFocusToVisible()
		}
		return m, nil
	case key.Matches(msg, k.Right):
		if len(m.board.Columns) > 0 {
			m.focusCol = (m.focusCol + 1) % len(m.board.Columns)
			m.focusTask = 0
			m.clampFocusToVisible()
		}
		return m, nil
	case key.Matches(msg, k.Up):
		m.stepFocusTask(-1)
		return m, nil
	case key.Matches(msg, k.Down):
		m.stepFocusTask(+1)
		return m, nil
	case key.Matches(msg, k.MoveLeft):
		return m.moveCardColumn(-1)
	case key.Matches(msg, k.MoveRight):
		return m.moveCardColumn(+1)
	case key.Matches(msg, k.MoveUp):
		return m.reorderCard(-1)
	case key.Matches(msg, k.MoveDown):
		return m.reorderCard(+1)
	case key.Matches(msg, k.New):
		m.beginEdit(true)
		return m, tea.Batch(m.titleInput.Focus(), textarea.Blink)
	case key.Matches(msg, k.Edit):
		if m.currentTask() != nil {
			m.beginEdit(false)
			return m, tea.Batch(m.titleInput.Focus(), textarea.Blink)
		}
	case key.Matches(msg, k.Delete):
		if m.currentTask() != nil {
			m.mode = modeConfirmDelete
		}
	}
	return m, nil
}

func (m *Model) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBoard
		m.filter = ""
		m.filterInput.Blur()
		m.clampFocusToVisible()
		return m, nil
	case "enter":
		m.filter = strings.TrimSpace(m.filterInput.Value())
		m.mode = modeBoard
		m.filterInput.Blur()
		m.clampFocusToVisible()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *Model) beginEdit(isNew bool) {
	m.mode = modeEdit
	m.editingNew = isNew
	if isNew {
		m.titleInput.SetValue("")
		m.bodyInput.SetValue("")
		m.labelsInput.SetValue("")
	} else {
		t := m.currentTask()
		m.titleInput.SetValue(t.Title)
		m.bodyInput.SetValue(t.Description)
		m.labelsInput.SetValue(strings.Join(t.Labels, ", "))
	}
	m.titleInput.Focus()
	m.bodyInput.Blur()
	m.labelsInput.Blur()
}

// cycleEditFocus moves keyboard focus through title → body → labels → title.
// shift==true reverses the cycle.
func (m *Model) cycleEditFocus(shift bool) {
	order := []func() bool{
		m.titleInput.Focused,
		m.bodyInput.Focused,
		m.labelsInput.Focused,
	}
	idx := 0
	for i, f := range order {
		if f() {
			idx = i
			break
		}
	}
	next := idx + 1
	if shift {
		next = idx - 1
	}
	next = (next + len(order)) % len(order)
	m.titleInput.Blur()
	m.bodyInput.Blur()
	m.labelsInput.Blur()
	switch next {
	case 0:
		m.titleInput.Focus()
	case 1:
		m.bodyInput.Focus()
	case 2:
		m.labelsInput.Focus()
	}
}

func (m *Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBoard
		return m, nil
	case "tab":
		m.cycleEditFocus(false)
		return m, nil
	case "shift+tab":
		m.cycleEditFocus(true)
		return m, nil
	case "ctrl+s":
		return m.commitEdit()
	}
	// Allow Enter to save only when a single-line field is focused (textarea
	// uses Enter for newline, so we must not hijack it there).
	if msg.String() == "enter" && (m.titleInput.Focused() || m.labelsInput.Focused()) {
		return m.commitEdit()
	}
	var cmd tea.Cmd
	switch {
	case m.titleInput.Focused():
		m.titleInput, cmd = m.titleInput.Update(msg)
	case m.labelsInput.Focused():
		m.labelsInput, cmd = m.labelsInput.Update(msg)
	default:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

// parseLabels splits the label field value (comma-separated) into a slice.
// Normalization (trim, lowercase, dedup) happens in tasks.normalizeLabels.
func parseLabels(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func labelsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *Model) commitEdit() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	body := m.bodyInput.Value()
	labels := parseLabels(m.labelsInput.Value())
	if title == "" {
		m.statusErr = true
		m.status = "title required"
		return m, nil
	}
	if m.saving {
		m.statusErr = true
		m.status = "save in progress; try again in a moment"
		return m, nil
	}
	var commitMsg string
	if m.editingNew {
		if _, err := m.board.AddTask(m.focusCol, title, body, labels); err != nil {
			m.statusErr = true
			m.status = err.Error()
			return m, nil
		}
		m.focusTask = len(m.board.Columns[m.focusCol].Tasks) - 1
		commitMsg = fmt.Sprintf("doit: create %q", title)
	} else {
		t := m.currentTask()
		oldTitle := t.Title
		// No-op short-circuit: if nothing changed, skip the save + commit so we
		// don't produce a noisy empty-edit commit. Compare normalized labels so
		// whitespace/case changes alone don't count as a change.
		normalizedLabels := tasks.NormalizeLabels(labels)
		if t.Title == title && t.Description == strings.TrimSpace(body) && labelsEqual(t.Labels, normalizedLabels) {
			m.mode = modeBoard
			m.statusErr = false
			m.status = "no changes"
			return m, nil
		}
		if err := m.board.UpdateTask(m.focusCol, m.focusTask, title, body, labels); err != nil {
			m.statusErr = true
			m.status = err.Error()
			return m, nil
		}
		commitMsg = fmt.Sprintf("doit: edit %q", oldTitle)
	}
	m.mode = modeBoard
	m.saving = true
	return m, m.saveCmd(commitMsg)
}

func (m *Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		t := m.currentTask()
		if t == nil {
			m.mode = modeBoard
			return m, nil
		}
		if m.saving {
			m.statusErr = true
			m.status = "save in progress; try again in a moment"
			m.mode = modeBoard
			return m, nil
		}
		title := t.Title
		if _, err := m.board.DeleteTask(m.focusCol, m.focusTask); err != nil {
			m.statusErr = true
			m.status = err.Error()
			m.mode = modeBoard
			return m, nil
		}
		m.clampFocus()
		m.mode = modeBoard
		m.saving = true
		return m, m.saveCmd(fmt.Sprintf("doit: delete %q", title))
	case "esc", "n", "N":
		m.mode = modeBoard
	}
	return m, nil
}

func (m *Model) moveCardColumn(delta int) (tea.Model, tea.Cmd) {
	if m.currentTask() == nil {
		return m, nil
	}
	target := m.focusCol + delta
	if target < 0 || target >= len(m.board.Columns) {
		return m, nil
	}
	t := m.currentTask()
	title := t.Title
	from := m.board.Columns[m.focusCol].Title
	to := m.board.Columns[target].Title
	newIdx, err := m.board.MoveTaskColumn(m.focusCol, m.focusTask, target)
	if err != nil {
		m.statusErr = true
		m.status = err.Error()
		return m, nil
	}
	m.focusCol = target
	m.focusTask = newIdx
	m.saving = true
	return m, m.saveCmd(fmt.Sprintf("doit: move %q from %s to %s", title, from, to))
}

func (m *Model) reorderCard(delta int) (tea.Model, tea.Cmd) {
	if m.currentTask() == nil {
		return m, nil
	}
	t := m.currentTask()
	title := t.Title
	newIdx, err := m.board.ReorderWithinColumn(m.focusCol, m.focusTask, delta)
	if err != nil {
		m.statusErr = true
		m.status = err.Error()
		return m, nil
	}
	if newIdx == m.focusTask {
		return m, nil
	}
	m.focusTask = newIdx
	m.saving = true
	return m, m.saveCmd(fmt.Sprintf("doit: reorder %q", title))
}

func (m *Model) clampFocus() {
	if len(m.board.Columns) == 0 {
		m.focusCol, m.focusTask = 0, 0
		return
	}
	if m.focusCol >= len(m.board.Columns) {
		m.focusCol = len(m.board.Columns) - 1
	}
	col := m.board.Columns[m.focusCol]
	if m.focusTask >= len(col.Tasks) {
		m.focusTask = len(col.Tasks) - 1
	}
	if m.focusTask < 0 {
		m.focusTask = 0
	}
}

// isDoneColumn reports whether a column holds "done" tasks, by title match.
// Case-insensitive and whitespace-trimmed so renamed variants still work.
func isDoneColumn(title string) bool {
	return strings.EqualFold(strings.TrimSpace(title), "done")
}

// isHiddenDone reports whether a task should be hidden by the time-based
// done-pruning rule. Requires a Done column, a configured horizon > 0, and
// showHidden == false.
func (m *Model) isHiddenDone(col tasks.Column, t tasks.Task) bool {
	if m.showHidden {
		return false
	}
	if m.cfg.HideDoneAfterDays <= 0 {
		return false
	}
	if !isDoneColumn(col.Title) {
		return false
	}
	age := time.Since(t.UpdatedAt)
	return age > time.Duration(m.cfg.HideDoneAfterDays)*24*time.Hour
}

// visibleTaskIndices returns the indices of tasks in the focused column that
// pass the current filter and aren't hidden by the done-pruning rule.
func (m *Model) visibleTaskIndices() []int {
	col := m.currentColumn()
	if col == nil {
		return nil
	}
	var out []int
	for i, t := range col.Tasks {
		if m.filter != "" && !matchesFilter(t, m.filter) {
			continue
		}
		if m.isHiddenDone(*col, t) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// stepFocusTask moves focus through the *visible* tasks of the current column,
// so j/k skip filtered-out cards instead of landing on invisible indices.
func (m *Model) stepFocusTask(delta int) {
	vis := m.visibleTaskIndices()
	if len(vis) == 0 {
		return
	}
	cur := -1
	for i, idx := range vis {
		if idx == m.focusTask {
			cur = i
			break
		}
	}
	if cur == -1 {
		// Current focus is hidden; snap to the first visible on any nav.
		m.focusTask = vis[0]
		return
	}
	next := cur + delta
	if next < 0 || next >= len(vis) {
		return
	}
	m.focusTask = vis[next]
}

// clampFocusToVisible snaps focus to the first visible task when the current
// focus would otherwise be hidden by the filter. Called after filter changes.
func (m *Model) clampFocusToVisible() {
	vis := m.visibleTaskIndices()
	if len(vis) == 0 {
		m.focusTask = 0
		return
	}
	for _, idx := range vis {
		if idx == m.focusTask {
			return
		}
	}
	m.focusTask = vis[0]
}

func (m *Model) currentColumn() *tasks.Column {
	if m.focusCol < 0 || m.focusCol >= len(m.board.Columns) {
		return nil
	}
	return &m.board.Columns[m.focusCol]
}

func (m *Model) currentTask() *tasks.Task {
	col := m.currentColumn()
	if col == nil || m.focusTask < 0 || m.focusTask >= len(col.Tasks) {
		return nil
	}
	return &col.Tasks[m.focusTask]
}

// -- View --

func (m *Model) View() string {
	switch m.mode {
	case modePromptCreate:
		return m.overlay(m.viewPromptCreate())
	case modeHelp:
		return m.overlay(m.viewHelpModal())
	case modeEdit:
		return m.overlay(m.viewEdit())
	case modeConfirmDelete:
		return m.overlay(m.viewConfirmDelete())
	case modeFilter:
		return m.viewBoardWith(m.viewFilterInline())
	}
	return m.viewBoard()
}

// overlay centers a modal in the viewport. The board is not dimmed behind it
// since lipgloss has no built-in opacity; we just render the modal on its own.
func (m *Model) overlay(content string) string {
	if m.width == 0 || m.height == 0 {
		return content
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) viewFilterInline() string {
	return styleStatusSegment.Render("/ " + m.filterInput.View())
}

func (m *Model) viewPromptCreate() string {
	body := fmt.Sprintf(
		"%s does not exist in %s.\n\nCreate it with default columns\n(Todo, In Progress, Done)?",
		m.cfg.File, m.cfg.Repo,
	)
	box := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render("Create board"),
		"",
		body,
		"",
		styleHelp.Render("[Y] create   [n] cancel"),
	)
	return styleModal.Render(box)
}

func (m *Model) viewHelpModal() string {
	rows := [][2]string{
		{"h/←  l/→", "move focus between columns"},
		{"j/↓  k/↑", "move focus between cards"},
		{"H  L", "move card to left/right column"},
		{"J  K", "reorder card within column"},
		{"n", "new card"},
		{"e / ⏎", "edit focused card"},
		{"d", "delete focused card"},
		{"/", "filter cards"},
		{"?", "toggle help"},
		{"q / Ctrl+C", "quit"},
	}
	keyStyle := lipgloss.NewStyle().Foreground(colPink).Bold(true).Width(12)
	descStyle := lipgloss.NewStyle().Foreground(colText)
	var lines []string
	lines = append(lines, styleModalTitle.Render("doit · keybindings"), "")
	for _, r := range rows {
		lines = append(lines, keyStyle.Render(r[0])+descStyle.Render(r[1]))
	}
	lines = append(lines,
		"",
		styleHelp.Render("In edit modal: Tab switches fields · ⏎/Ctrl+S saves · Esc cancels"),
		"",
		styleHelp.Render("press any key to return"),
	)
	return styleModal.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) viewEdit() string {
	title := "Edit task"
	if m.editingNew {
		title = "New task"
	}
	labelStyle := lipgloss.NewStyle().Foreground(colMuted).MarginTop(1)
	box := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render(title),
		labelStyle.Render("Title"),
		m.titleInput.View(),
		labelStyle.Render("Body"),
		m.bodyInput.View(),
		labelStyle.Render("Labels (comma-separated)"),
		m.labelsInput.View(),
		"",
		styleHelp.Render("Tab/Shift+Tab cycle · ⏎ (title/labels) / Ctrl+S save · Esc cancel"),
	)
	return styleModal.Render(box)
}

func (m *Model) viewConfirmDelete() string {
	t := m.currentTask()
	if t == nil {
		return styleModal.Render("nothing to delete")
	}
	box := lipgloss.JoinVertical(lipgloss.Left,
		styleModalTitle.Render("Delete task"),
		"",
		fmt.Sprintf("Delete %q?", t.Title),
		"",
		styleHelp.Render("[y] yes   [n/Esc] no"),
	)
	return styleModal.Render(box)
}

// viewBoard renders the main Kanban layout: header accent bar, columns, and
// a vim-style status footer.
func (m *Model) viewBoard() string {
	return m.viewBoardWith("")
}

// viewBoardWith lets callers substitute a custom center segment into the
// status bar (e.g. an inline filter input).
func (m *Model) viewBoardWith(centerOverride string) string {
	if len(m.board.Columns) == 0 {
		return lipgloss.NewStyle().Padding(1, 2).Render(
			styleBoardTitle.Render(m.board.Title) + "\n\n" +
				styleHelp.Render("Board has no columns. Edit "+m.cfg.File+" to add some."),
		)
	}

	header := m.renderHeader()

	var body string
	if paneW := m.detailPaneWidth(); paneW > 0 {
		cols := m.renderColumnsWidth(m.width - paneW)
		detail := m.renderDetailPane(paneW)
		body = lipgloss.JoinHorizontal(lipgloss.Top, cols, detail)
	} else {
		body = m.renderColumns()
	}

	statusBar := m.renderStatusBar(centerOverride)
	helpLine := m.help.View(m.keymap)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", statusBar, helpLine)
}

func (m *Model) renderHeader() string {
	bar := styleAccentBar.Render("▍")
	title := styleBoardTitle.Render(m.board.Title)
	file := styleFilename.Render(" · " + m.cfg.File)
	return lipgloss.NewStyle().Padding(0, 1).Render(bar + " " + title + file)
}

func (m *Model) renderColumns() string {
	return m.renderColumnsWidth(m.width)
}

func (m *Model) renderColumnsWidth(total int) string {
	colCount := len(m.board.Columns)
	gutters := 2 * colCount
	colWidth := (total - gutters) / colCount
	if colWidth < 18 {
		colWidth = 18
	}
	var rendered []string
	for ci, col := range m.board.Columns {
		rendered = append(rendered, m.renderColumn(ci, col, colWidth))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m *Model) renderColumn(ci int, col tasks.Column, width int) string {
	focused := ci == m.focusCol
	titleStyle := styleColumnTitle
	ruleStyle := styleColumnRule
	if focused {
		titleStyle = styleColumnTitleFocused
		ruleStyle = styleColumnRuleFocused
	}
	title := titleStyle.Render(col.Title) + " " + styleColumnCount.Render(fmt.Sprintf("· %d", len(col.Tasks)))
	rule := ruleStyle.Render(strings.Repeat("─", width-2))

	parts := []string{title, rule}
	hidden := 0
	for ti, t := range col.Tasks {
		if m.filter != "" && !matchesFilter(t, m.filter) {
			continue
		}
		if m.isHiddenDone(col, t) {
			hidden++
			continue
		}
		parts = append(parts, m.renderCard(ci, ti, t, width-2))
	}
	if len(parts) == 2 {
		parts = append(parts, styleHelp.Render("  (empty)"))
	}
	if hidden > 0 {
		parts = append(parts, styleHelp.Render(fmt.Sprintf("  +%d hidden · press a", hidden)))
	}
	return lipgloss.NewStyle().Width(width).Padding(0, 1).Render(
		lipgloss.JoinVertical(lipgloss.Left, parts...),
	)
}

func (m *Model) renderCard(ci, ti int, t tasks.Task, width int) string {
	focused := ci == m.focusCol && ti == m.focusTask

	stripeChar := "▍"
	var stripe string
	if focused {
		stripe = lipgloss.NewStyle().Foreground(stripeColorFor(t.Labels)).Render(stripeChar)
	} else if len(t.Labels) > 0 {
		// A dimmer stripe keyed off the tag color so users can scan by type
		// even when the card isn't focused.
		stripe = lipgloss.NewStyle().Foreground(stripeColorFor(t.Labels)).Faint(true).Render(stripeChar)
	} else {
		stripe = styleCardStripeIdle.Render(stripeChar)
	}

	titleStyle := styleCardTitle
	if focused {
		titleStyle = styleCardTitleFocused
	}

	innerWidth := width - 2
	if innerWidth < 4 {
		innerWidth = 4
	}

	title := titleStyle.Render(truncate(t.Title, innerWidth))

	var lines []string
	lines = append(lines, title)
	if t.Description != "" {
		preview := strings.SplitN(t.Description, "\n", 2)[0]
		lines = append(lines, styleCardPreview.Render(truncate(preview, innerWidth)))
	}
	if len(t.Labels) > 0 {
		var pills []string
		for _, l := range t.Labels {
			pills = append(pills, labelPill(l))
		}
		lines = append(lines, strings.Join(pills, " "))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	card := lipgloss.JoinHorizontal(lipgloss.Top, stripe, " "+body)
	return lipgloss.NewStyle().MarginBottom(1).Render(card)
}

func (m *Model) renderDetailPane(width int) string {
	pane := lipgloss.NewStyle().
		Width(width).
		Padding(0, 2).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(colFaint)

	t := m.currentTask()
	if t == nil {
		return pane.Render(styleHelp.Render("No task selected."))
	}

	stripe := lipgloss.NewStyle().Foreground(stripeColorFor(t.Labels)).Render("▍")
	title := stripe + " " + styleCardTitleFocused.Render(t.Title)

	var pills string
	if len(t.Labels) > 0 {
		var p []string
		for _, l := range t.Labels {
			p = append(p, labelPill(l))
		}
		pills = strings.Join(p, "  ")
	}

	var body string
	if t.Description == "" {
		body = styleHelp.Render("(no description — press e to edit)")
	} else if m.renderer != nil {
		out, err := m.renderer.Render(t.Description)
		if err != nil {
			body = t.Description
		} else {
			body = strings.TrimRight(out, "\n")
		}
	} else {
		body = t.Description
	}

	meta := styleHelp.Render(fmt.Sprintf("id:%s · updated %s", t.ID, t.UpdatedAt.Format("2006-01-02 15:04")))

	parts := []string{title}
	if pills != "" {
		parts = append(parts, pills)
	}
	parts = append(parts, "", body, "", meta)
	return pane.Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func (m *Model) renderStatusBar(centerOverride string) string {
	mode := styleStatusMode.Render("BOARD")
	if m.saving {
		mode = styleStatusSaving.Render(m.spinner.View() + " SAVE")
	}
	if m.statusErr {
		mode = styleStatusError.Render("ERROR")
	}

	left := mode + styleStatusSegment.Render(m.cfg.File)

	center := centerOverride
	if center == "" && m.status != "" {
		s := m.status
		if m.statusErr {
			s = "× " + s
		}
		center = styleStatusSegment.Render(truncate(s, 60))
	}

	var pos string
	if col := m.currentColumn(); col != nil {
		pos = fmt.Sprintf("%s %d/%d · %d/%d",
			col.Title,
			m.focusCol+1, len(m.board.Columns),
			min(m.focusTask+1, len(col.Tasks)), len(col.Tasks),
		)
	}
	right := styleStatusSegmentMuted.Render(pos)
	if m.filter != "" {
		right = styleStatusSegment.Render("/"+m.filter) + right
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(center) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	filler := styleStatusBarBase.Render(strings.Repeat(" ", gap))
	return styleStatusBarBase.Width(m.width).Render(left + center + filler + right)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	return string(r[:max-1]) + "…"
}

func matchesFilter(t tasks.Task, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(t.Title), q) ||
		strings.Contains(strings.ToLower(t.Description), q)
}
