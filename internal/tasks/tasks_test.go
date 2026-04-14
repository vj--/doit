package tasks

import "testing"

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Todo":        "todo",
		"In Progress": "in-progress",
		"  Done!!! ":  "done",
		"":            "",
		"A / B":       "a-b",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMoveTaskColumn(t *testing.T) {
	b := DefaultBoard()
	if _, err := b.AddTask(0, "one", "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AddTask(0, "two", "", nil); err != nil {
		t.Fatal(err)
	}
	newIdx, err := b.MoveTaskColumn(0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if newIdx != 0 {
		t.Errorf("newIdx = %d, want 0", newIdx)
	}
	if len(b.Columns[0].Tasks) != 1 || b.Columns[0].Tasks[0].Title != "two" {
		t.Errorf("src column wrong: %+v", b.Columns[0].Tasks)
	}
	if len(b.Columns[1].Tasks) != 1 || b.Columns[1].Tasks[0].Title != "one" {
		t.Errorf("dst column wrong: %+v", b.Columns[1].Tasks)
	}
}

func TestReorderWithinColumn(t *testing.T) {
	b := DefaultBoard()
	for _, title := range []string{"a", "b", "c"} {
		if _, err := b.AddTask(0, title, "", nil); err != nil {
			t.Fatal(err)
		}
	}
	newIdx, err := b.ReorderWithinColumn(0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if newIdx != 1 {
		t.Errorf("newIdx = %d, want 1", newIdx)
	}
	got := []string{b.Columns[0].Tasks[0].Title, b.Columns[0].Tasks[1].Title, b.Columns[0].Tasks[2].Title}
	want := []string{"b", "a", "c"}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("order = %v, want %v", got, want)
			break
		}
	}
}

func TestReorderBounds(t *testing.T) {
	b := DefaultBoard()
	if _, err := b.AddTask(0, "a", "", nil); err != nil {
		t.Fatal(err)
	}
	if idx, _ := b.ReorderWithinColumn(0, 0, -1); idx != 0 {
		t.Errorf("expected clamp at 0, got %d", idx)
	}
	if idx, _ := b.ReorderWithinColumn(0, 0, 5); idx != 0 {
		t.Errorf("expected clamp at 0, got %d", idx)
	}
}
