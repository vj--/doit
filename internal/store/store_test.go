package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vj--/doit/internal/tasks"
)

type fakeGit struct {
	called []string
}

func (f *fakeGit) AddCommit(ctx context.Context, paths []string, msg string) (bool, error) {
	f.called = append(f.called, msg)
	return true, nil
}

func TestSaveAtomicAndCommit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "board.md")
	g := &fakeGit{}
	s := New(path, g, false)

	b := tasks.DefaultBoard()
	if _, err := b.AddTask(0, "first", "", nil); err != nil {
		t.Fatal(err)
	}

	committed, err := s.Save(context.Background(), b, `doit: create "first"`)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !committed {
		t.Errorf("expected committed=true")
	}
	if len(g.called) != 1 || g.called[0] == "" {
		t.Errorf("unexpected commit calls: %v", g.called)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
}

func TestSaveNoCommitSkipsGit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "board.md")
	g := &fakeGit{}
	s := New(path, g, true)
	b := tasks.DefaultBoard()
	committed, err := s.Save(context.Background(), b, "msg")
	if err != nil {
		t.Fatal(err)
	}
	if committed {
		t.Errorf("expected committed=false in no-commit mode")
	}
	if len(g.called) != 0 {
		t.Errorf("git must not be called in no-commit mode, got %v", g.called)
	}
}
