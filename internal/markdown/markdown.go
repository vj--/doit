// Package markdown parses and renders the board markdown format defined in CLAUDE.md.
//
// Schema:
//
//	# Board Title
//
//	## Column Name
//
//	- <!-- id:<id> labels:a,b created:<RFC3339> updated:<RFC3339> -->
//	  **Task title**
//	  Optional body line 1
//	  Optional body line 2
//
// Columns are level-2 headings in document order; tasks are top-level list
// items under each column. Metadata lives in an HTML comment on the first
// line of each list item so it does not render in GFM previews.
package markdown

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vj/doit/internal/tasks"
)

// Parse turns the markdown bytes into a Board.
func Parse(data []byte) (tasks.Board, error) {
	b := tasks.Board{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return b, err
	}

	i := 0
	// Title.
	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			b.Title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			i++
			break
		}
		break
	}

	var curCol *tasks.Column
	flushCol := func() {
		if curCol != nil {
			b.Columns = append(b.Columns, *curCol)
			curCol = nil
		}
	}

	for i < len(lines) {
		raw := lines[i]

		// Column heading. Match on the raw line (column 0) so nested "## " inside
		// a fenced code block inside a task body doesn't falsely close the column.
		if strings.HasPrefix(raw, "## ") {
			flushCol()
			title := strings.TrimSpace(strings.TrimPrefix(raw, "## "))
			curCol = &tasks.Column{ID: tasks.Slug(title), Title: title}
			i++
			continue
		}

		// Task: list bullet at column 0. Indented "- " lines (GFM task checklists,
		// nested lists) belong to a body and must NOT start a new task.
		if curCol != nil && strings.HasPrefix(raw, "- ") {
			item := []string{strings.TrimPrefix(raw, "- ")}
			i++
			// Collect continuation lines: anything indented (treat as body line,
			// with leading whitespace stripped) OR blank (preserved as blank). A
			// non-indented, non-blank line ends the item — that means a new
			// column heading or a new "- " bullet at column 0.
			for i < len(lines) {
				next := lines[i]
				if next == "" {
					item = append(item, "")
					i++
					continue
				}
				if strings.HasPrefix(next, " ") || strings.HasPrefix(next, "\t") {
					item = append(item, strings.TrimLeft(next, " \t"))
					i++
					continue
				}
				break
			}
			// Drop trailing blank lines that come from the paragraph break between
			// tasks; they are formatting, not body content.
			for len(item) > 0 && item[len(item)-1] == "" {
				item = item[:len(item)-1]
			}
			t, err := parseTask(item)
			if err != nil {
				return b, fmt.Errorf("parse task in column %q: %w", curCol.Title, err)
			}
			curCol.Tasks = append(curCol.Tasks, t)
			continue
		}

		i++
	}
	flushCol()
	return b, nil
}

var metaRe = regexp.MustCompile(`^<!--\s*(.*?)\s*-->\s*$`)

func parseTask(lines []string) (tasks.Task, error) {
	var t tasks.Task
	if len(lines) == 0 {
		return t, fmt.Errorf("empty task")
	}
	idx := 0
	// Optional metadata comment on first line.
	if m := metaRe.FindStringSubmatch(lines[idx]); m != nil {
		parseMeta(&t, m[1])
		idx++
	}
	if idx >= len(lines) {
		return t, fmt.Errorf("task missing title")
	}
	// Title line: **Title** or plain text.
	title := strings.TrimSpace(lines[idx])
	title = strings.TrimPrefix(title, "**")
	title = strings.TrimSuffix(title, "**")
	t.Title = strings.TrimSpace(title)
	idx++
	// Remaining lines are body.
	if idx < len(lines) {
		t.Description = strings.TrimSpace(strings.Join(lines[idx:], "\n"))
	}
	if t.ID == "" {
		t.ID = tasks.NewID()
	}
	return t, nil
}

// parseMeta reads `key:value` pairs separated by whitespace.
func parseMeta(t *tasks.Task, s string) {
	for _, tok := range strings.Fields(s) {
		k, v, ok := strings.Cut(tok, ":")
		if !ok {
			continue
		}
		switch k {
		case "id":
			t.ID = v
		case "labels":
			if v != "" {
				t.Labels = strings.Split(v, ",")
			}
		case "created":
			if tm, err := time.Parse(time.RFC3339, v); err == nil {
				t.CreatedAt = tm
			}
		case "updated":
			if tm, err := time.Parse(time.RFC3339, v); err == nil {
				t.UpdatedAt = tm
			}
		}
	}
}

// Render serializes a Board back to markdown.
func Render(b tasks.Board) []byte {
	var sb strings.Builder
	title := b.Title
	if title == "" {
		title = "My Board"
	}
	fmt.Fprintf(&sb, "# %s\n\n", title)
	for _, c := range b.Columns {
		fmt.Fprintf(&sb, "## %s\n\n", c.Title)
		for _, t := range c.Tasks {
			sb.WriteString("- ")
			sb.WriteString(renderMeta(t))
			sb.WriteByte('\n')
			fmt.Fprintf(&sb, "  **%s**\n", escapeInline(t.Title))
			if t.Description != "" {
				for _, line := range strings.Split(t.Description, "\n") {
					fmt.Fprintf(&sb, "  %s\n", line)
				}
			}
			sb.WriteByte('\n')
		}
	}
	return []byte(sb.String())
}

func renderMeta(t tasks.Task) string {
	var parts []string
	if t.ID != "" {
		parts = append(parts, "id:"+t.ID)
	}
	if len(t.Labels) > 0 {
		parts = append(parts, "labels:"+strings.Join(t.Labels, ","))
	}
	if !t.CreatedAt.IsZero() {
		parts = append(parts, "created:"+t.CreatedAt.UTC().Format(time.RFC3339))
	}
	if !t.UpdatedAt.IsZero() {
		parts = append(parts, "updated:"+t.UpdatedAt.UTC().Format(time.RFC3339))
	}
	return "<!-- " + strings.Join(parts, " ") + " -->"
}

func escapeInline(s string) string {
	// Escape any literal "**" so it doesn't terminate the bold title wrapper.
	// Backslash-escaping each asterisk preserves the title verbatim and round-trips
	// through Parse (which strips the surrounding ** and trims whitespace).
	return strings.ReplaceAll(s, "**", `\*\*`)
}
