package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

	titleInput  textinput.Model
	bodyInput   textarea.Model
	filterInput textinput.Model
	editingNew  bool

	status    string
	statusErr bool

	saving bool
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
	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "Filter (substring)"
	m.filterInput.CharLimit = 200

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

func (m *Model) Init() tea.Cmd { return tickReloadCmd() }

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
		return m, nil

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
	} else {
		t := m.currentTask()
		m.titleInput.SetValue(t.Title)
		m.bodyInput.SetValue(t.Description)
	}
	m.titleInput.Focus()
	m.bodyInput.Blur()
}

func (m *Model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBoard
		return m, nil
	case "tab":
		if m.titleInput.Focused() {
			m.titleInput.Blur()
			m.bodyInput.Focus()
		} else {
			m.bodyInput.Blur()
			m.titleInput.Focus()
		}
		return m, nil
	case "ctrl+s":
		return m.commitEdit()
	}
	// Allow Enter to save only when title field is focused (textarea uses Enter for newline).
	if msg.String() == "enter" && m.titleInput.Focused() {
		return m.commitEdit()
	}
	var cmd tea.Cmd
	if m.titleInput.Focused() {
		m.titleInput, cmd = m.titleInput.Update(msg)
	} else {
		m.bodyInput, cmd = m.bodyInput.Update(msg)
	}
	return m, cmd
}

func (m *Model) commitEdit() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(m.titleInput.Value())
	body := m.bodyInput.Value()
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
		if _, err := m.board.AddTask(m.focusCol, title, body); err != nil {
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
		// don't produce a noisy empty-edit commit.
		if t.Title == title && t.Description == strings.TrimSpace(body) {
			m.mode = modeBoard
			m.statusErr = false
			m.status = "no changes"
			return m, nil
		}
		if err := m.board.UpdateTask(m.focusCol, m.focusTask, title, body); err != nil {
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

// visibleTaskIndices returns the indices of tasks in the focused column that
// pass the current filter. When no filter is set, it returns all indices.
func (m *Model) visibleTaskIndices() []int {
	col := m.currentColumn()
	if col == nil {
		return nil
	}
	var out []int
	for i, t := range col.Tasks {
		if m.filter == "" || matchesFilter(t, m.filter) {
			out = append(out, i)
		}
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
		return m.viewPromptCreate()
	case modeHelp:
		return m.viewHelp()
	case modeEdit:
		return m.viewEdit()
	case modeConfirmDelete:
		return m.viewConfirmDelete()
	case modeFilter:
		return m.viewFilter()
	}
	return m.viewBoard()
}

func (m *Model) viewFilter() string {
	box := lipgloss.JoinVertical(lipgloss.Left,
		styleHeader.Render("Filter"),
		"",
		m.filterInput.View(),
		"",
		styleHelp.Render("Enter: apply   Esc: clear"),
	)
	return lipgloss.NewStyle().Padding(1, 2).Render(box)
}

func (m *Model) viewPromptCreate() string {
	msg := fmt.Sprintf(
		"%s does not exist in %s.\n\nCreate it with default columns (Todo, In Progress, Done)? [Y/n]",
		m.cfg.File, m.cfg.Repo,
	)
	return lipgloss.NewStyle().Padding(1, 2).Render(msg)
}

func (m *Model) viewHelp() string {
	lines := []string{
		"doit — keybindings",
		"",
		"  h/←, l/→    move focus between columns",
		"  j/↓, k/↑    move focus between cards",
		"  H, L        move focused card to column left/right",
		"  J, K        reorder focused card within column",
		"  n           new card",
		"  e / Enter   edit focused card",
		"  d           delete focused card (with confirm)",
		"  /           filter cards",
		"  ?           toggle help",
		"  q / Ctrl+C  quit",
		"",
		"In edit modal: Tab switches title/body, Enter (on title) or Ctrl+S saves, Esc cancels.",
		"",
		"press any key to return",
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) viewEdit() string {
	title := "Edit task"
	if m.editingNew {
		title = "New task"
	}
	box := lipgloss.JoinVertical(lipgloss.Left,
		styleHeader.Render(title),
		"",
		"Title:",
		m.titleInput.View(),
		"",
		"Body:",
		m.bodyInput.View(),
		"",
		styleHelp.Render("Tab: switch field   Enter (title) / Ctrl+S: save   Esc: cancel"),
	)
	return lipgloss.NewStyle().Padding(1, 2).Render(box)
}

func (m *Model) viewConfirmDelete() string {
	t := m.currentTask()
	if t == nil {
		return "nothing to delete"
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(
		fmt.Sprintf("Delete %q? [y/N]", t.Title),
	)
}

func (m *Model) viewBoard() string {
	if len(m.board.Columns) == 0 {
		return lipgloss.NewStyle().Padding(1, 2).Render("Board has no columns. Edit board.md to add some.")
	}

	header := styleHeader.Render(m.board.Title) + " " +
		styleStatus.Render(fmt.Sprintf("(%s)", m.cfg.File))

	colCount := len(m.board.Columns)
	gutters := 2 * colCount // margin per column
	colWidth := (m.width - gutters) / colCount
	if colWidth < 20 {
		colWidth = 20
	}

	var rendered []string
	for ci, col := range m.board.Columns {
		rendered = append(rendered, m.renderColumn(ci, col, colWidth))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, rendered...)

	status := m.status
	if m.saving {
		status = "saving…"
	}
	statusLine := styleStatus.Render(status)
	if m.statusErr {
		statusLine = styleError.Render(status)
	}
	hint := styleHelp.Render("? help   n new   e edit   H/L move   q quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusLine, hint)
}

func (m *Model) renderColumn(ci int, col tasks.Column, width int) string {
	title := fmt.Sprintf("%s (%d)", col.Title, len(col.Tasks))
	var cards []string
	cards = append(cards, styleColumnTitle.Render(title))
	for ti, t := range col.Tasks {
		if m.filter != "" && !matchesFilter(t, m.filter) {
			continue
		}
		cards = append(cards, m.renderCard(ci, ti, t, width-4))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, cards...)
	style := styleColumn
	if ci == m.focusCol {
		style = styleColumnFocused
	}
	return style.Width(width).Render(content)
}

func (m *Model) renderCard(ci, ti int, t tasks.Task, width int) string {
	body := t.Title
	if t.Description != "" {
		// Show first line of description as a subtle preview.
		preview := strings.SplitN(t.Description, "\n", 2)[0]
		max := width - 2
		if max < 1 {
			max = 1
		}
		runes := []rune(preview)
		if len(runes) > max {
			if max-1 < 0 {
				preview = "…"
			} else {
				preview = string(runes[:max-1]) + "…"
			}
		}
		body += "\n" + styleHelp.Render(preview)
	}
	style := styleCard
	if ci == m.focusCol && ti == m.focusTask {
		style = styleCardFocused
	}
	return style.Width(width).Render(body)
}

func matchesFilter(t tasks.Task, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(t.Title), q) ||
		strings.Contains(strings.ToLower(t.Description), q)
}
