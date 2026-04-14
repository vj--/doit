// Package tasks defines the Kanban domain model and pure operations on it.
package tasks

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Board struct {
	Title   string
	Columns []Column
}

type Column struct {
	ID    string
	Title string
	Tasks []Task
}

type Task struct {
	ID          string
	Title       string
	Description string
	Labels      []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DefaultBoard returns an empty board with the conventional columns.
func DefaultBoard() Board {
	return Board{
		Title: "My Board",
		Columns: []Column{
			{ID: "todo", Title: "Todo"},
			{ID: "in-progress", Title: "In Progress"},
			{ID: "done", Title: "Done"},
		},
	}
}

// NewID returns a short random hex id suitable as a stable task id.
func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand shouldn't fail; fall back to timestamp
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// Slug produces a lowercase, hyphenated id from a column title.
func Slug(s string) string {
	var b strings.Builder
	prevHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		default:
			if !prevHyphen && b.Len() > 0 {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func (b *Board) ColumnIndex(id string) (int, bool) {
	for i, c := range b.Columns {
		if c.ID == id {
			return i, true
		}
	}
	return -1, false
}

func (b *Board) FindTask(taskID string) (colIdx, taskIdx int, ok bool) {
	for ci, c := range b.Columns {
		for ti, t := range c.Tasks {
			if t.ID == taskID {
				return ci, ti, true
			}
		}
	}
	return -1, -1, false
}

// AddTask appends a new task to the given column.
func (b *Board) AddTask(colIdx int, title, description string) (Task, error) {
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return Task{}, errors.New("column index out of range")
	}
	now := time.Now().UTC().Truncate(time.Second)
	t := Task{
		ID:          NewID(),
		Title:       strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if t.Title == "" {
		return Task{}, errors.New("task title must not be empty")
	}
	b.Columns[colIdx].Tasks = append(b.Columns[colIdx].Tasks, t)
	return t, nil
}

// UpdateTask replaces the title/description of a task, preserving other fields.
func (b *Board) UpdateTask(colIdx, taskIdx int, title, description string) error {
	if err := b.checkPos(colIdx, taskIdx); err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("task title must not be empty")
	}
	t := &b.Columns[colIdx].Tasks[taskIdx]
	t.Title = title
	t.Description = strings.TrimSpace(description)
	t.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	return nil
}

// DeleteTask removes a task at (colIdx, taskIdx).
func (b *Board) DeleteTask(colIdx, taskIdx int) (Task, error) {
	if err := b.checkPos(colIdx, taskIdx); err != nil {
		return Task{}, err
	}
	col := &b.Columns[colIdx]
	t := col.Tasks[taskIdx]
	col.Tasks = append(col.Tasks[:taskIdx], col.Tasks[taskIdx+1:]...)
	return t, nil
}

// MoveTaskColumn moves a task from (fromCol, fromIdx) to the end of toCol.
// Returns the new task index in the destination column.
func (b *Board) MoveTaskColumn(fromCol, fromIdx, toCol int) (int, error) {
	if err := b.checkPos(fromCol, fromIdx); err != nil {
		return -1, err
	}
	if toCol < 0 || toCol >= len(b.Columns) {
		return -1, errors.New("destination column out of range")
	}
	if toCol == fromCol {
		return fromIdx, nil
	}
	src := &b.Columns[fromCol]
	t := src.Tasks[fromIdx]
	src.Tasks = append(src.Tasks[:fromIdx], src.Tasks[fromIdx+1:]...)
	dst := &b.Columns[toCol]
	dst.Tasks = append(dst.Tasks, t)
	return len(dst.Tasks) - 1, nil
}

// ReorderWithinColumn moves a task up (-1) or down (+1) within its column.
func (b *Board) ReorderWithinColumn(colIdx, taskIdx, delta int) (int, error) {
	if err := b.checkPos(colIdx, taskIdx); err != nil {
		return -1, err
	}
	col := &b.Columns[colIdx]
	newIdx := taskIdx + delta
	if newIdx < 0 || newIdx >= len(col.Tasks) {
		return taskIdx, nil
	}
	col.Tasks[taskIdx], col.Tasks[newIdx] = col.Tasks[newIdx], col.Tasks[taskIdx]
	return newIdx, nil
}

func (b *Board) checkPos(colIdx, taskIdx int) error {
	if colIdx < 0 || colIdx >= len(b.Columns) {
		return errors.New("column index out of range")
	}
	if taskIdx < 0 || taskIdx >= len(b.Columns[colIdx].Tasks) {
		return errors.New("task index out of range")
	}
	return nil
}
