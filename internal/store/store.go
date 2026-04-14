// Package store is the file-backed repository for the board markdown.
//
// Responsibilities:
//   - Load the markdown file into a Board (or report "not exists").
//   - Save a Board with atomic write (tmp + rename), then perform a
//     per-action git commit via the injected git.Client (unless NoCommit).
//   - Track mtime so the caller can detect external edits.
package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vj/doit/internal/markdown"
	"github.com/vj/doit/internal/tasks"
)

type GitClient interface {
	AddCommit(ctx context.Context, paths []string, msg string) (bool, error)
}

type Store struct {
	mu       sync.Mutex
	path     string
	git      GitClient
	noCommit bool
	lastMod  time.Time
}

func New(path string, git GitClient, noCommit bool) *Store {
	return &Store{path: path, git: git, noCommit: noCommit}
}

func (s *Store) Path() string { return s.path }

// Exists reports whether the markdown file is present on disk.
func (s *Store) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Load reads and parses the file. Returns os.ErrNotExist if missing.
func (s *Store) Load() (tasks.Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) loadLocked() (tasks.Board, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return tasks.Board{}, err
	}
	info, err := os.Stat(s.path)
	if err == nil {
		s.lastMod = info.ModTime()
	}
	return markdown.Parse(data)
}

// ExternallyModified reports whether the file's mtime is newer than the last
// value observed by Load or Save.
func (s *Store) ExternallyModified() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := os.Stat(s.path)
	if err != nil {
		return false, err
	}
	return info.ModTime().After(s.lastMod), nil
}

// Reload re-reads and re-parses the file unconditionally.
func (s *Store) Reload() (tasks.Board, error) { return s.Load() }

// Save writes the board to disk atomically and commits with msg.
// Returns whether a commit was actually made (false if --no-commit or if
// there was nothing staged).
func (s *Store) Save(ctx context.Context, b tasks.Board, commitMsg string) (committed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writeAtomic(markdown.Render(b)); err != nil {
		return false, err
	}
	info, err := os.Stat(s.path)
	if err == nil {
		s.lastMod = info.ModTime()
	}
	if s.noCommit {
		return false, nil
	}
	if s.git == nil {
		return false, errors.New("git client not configured")
	}
	return s.git.AddCommit(ctx, []string{filepath.Base(s.path)}, commitMsg)
}

func (s *Store) writeAtomic(data []byte) error {
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("rename tmp: %w", err)
	}
	cleanup = false
	return nil
}
