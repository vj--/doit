package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Repo     string
	File     string
	NoCommit bool
}

func (c Config) FilePath() string {
	return filepath.Join(c.Repo, c.File)
}

func (c Config) Validate() error {
	if c.Repo == "" {
		return errors.New("--repo is required")
	}
	info, err := os.Stat(c.Repo)
	if err != nil {
		return fmt.Errorf("repo path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("repo path %q is not a directory", c.Repo)
	}
	// Must be a git working tree (contains a .git directory or file).
	gitPath := filepath.Join(c.Repo, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return fmt.Errorf("%q is not a git repository (no .git found)", c.Repo)
	}
	if c.File == "" {
		return errors.New("--file must not be empty")
	}
	if filepath.IsAbs(c.File) || filepath.Clean(c.File) != c.File || containsDotDot(c.File) {
		return fmt.Errorf("--file must be a simple relative name, got %q", c.File)
	}
	return nil
}

func containsDotDot(p string) bool {
	// Walk path segments using the OS path separator (filepath.SplitList splits
	// the PATH list separator, which is the wrong API here).
	clean := filepath.ToSlash(filepath.Clean(p))
	for _, seg := range strings.Split(clean, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}
