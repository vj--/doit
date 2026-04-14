package markdown

import (
	"strings"
	"testing"
	"time"

	"github.com/vj/doit/internal/tasks"
)

func TestRoundTrip(t *testing.T) {
	orig := tasks.Board{
		Title: "Project",
		Columns: []tasks.Column{
			{
				ID: "todo", Title: "Todo",
				Tasks: []tasks.Task{
					{
						ID:          "abc123",
						Title:       "Fix login",
						Description: "Users land on a blank page.\nSecond line.",
						Labels:      []string{"bug", "urgent"},
						CreatedAt:   time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
						UpdatedAt:   time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			{ID: "done", Title: "Done"},
		},
	}
	out := Render(orig)
	parsed, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if parsed.Title != orig.Title {
		t.Errorf("title: got %q, want %q", parsed.Title, orig.Title)
	}
	if len(parsed.Columns) != len(orig.Columns) {
		t.Fatalf("columns: got %d, want %d (output: %s)", len(parsed.Columns), len(orig.Columns), out)
	}
	got := parsed.Columns[0].Tasks[0]
	want := orig.Columns[0].Tasks[0]
	if got.ID != want.ID || got.Title != want.Title {
		t.Errorf("task id/title mismatch: got %+v, want %+v", got, want)
	}
	if got.Description != want.Description {
		t.Errorf("description:\ngot  %q\nwant %q", got.Description, want.Description)
	}
	if strings.Join(got.Labels, ",") != strings.Join(want.Labels, ",") {
		t.Errorf("labels: got %v, want %v", got.Labels, want.Labels)
	}
}

func TestParseEmpty(t *testing.T) {
	b, err := Parse([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 0 {
		t.Errorf("expected no columns, got %d", len(b.Columns))
	}
}
