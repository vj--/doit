// Package git is a thin, allow-listed wrapper around the system git binary.
//
// Only the subcommands in allowedSubcommands may be executed. Attempting
// any other subcommand (push, pull, fetch, reset, etc.) returns an error.
// This is enforced in exec() and verified by tests.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// allowedSubcommands is the complete set of git subcommands this package may run.
// Any change to this list must be code-reviewed and reflected in CLAUDE.md.
var allowedSubcommands = map[string]struct{}{
	"rev-parse": {},
	"status":    {},
	"log":       {},
	"add":       {},
	"commit":    {},
	"config":    {},
}

type Client struct {
	repo string
}

func New(repo string) *Client {
	return &Client{repo: repo}
}

// IsRepo reports whether the configured path is a git working tree.
func (c *Client) IsRepo(ctx context.Context) bool {
	_, err := c.exec(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// AddCommit stages the given paths and records a commit with msg.
// If there are no staged changes after `git add`, it returns (false, nil).
func (c *Client) AddCommit(ctx context.Context, paths []string, msg string) (committed bool, err error) {
	if strings.TrimSpace(msg) == "" {
		return false, errors.New("commit message must not be empty")
	}
	addArgs := append([]string{"--"}, paths...)
	if _, err := c.exec(ctx, "add", addArgs...); err != nil {
		return false, fmt.Errorf("git add: %w", err)
	}
	// Check if there's anything staged.
	out, err := c.exec(ctx, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	if !hasStagedChange(out) {
		return false, nil
	}
	if _, err := c.exec(ctx, "commit", "-m", msg); err != nil {
		return false, fmt.Errorf("git commit: %w", err)
	}
	return true, nil
}

// LastCommitForFile returns "<short-sha> <subject>" for the most recent commit touching path.
func (c *Client) LastCommitForFile(ctx context.Context, path string) (string, error) {
	out, err := c.exec(ctx, "log", "-1", "--pretty=format:%h %s", "--", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func hasStagedChange(porcelain string) bool {
	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 2 {
			continue
		}
		// Index status is the first char; space means no staged change.
		if line[0] != ' ' && line[0] != '?' {
			return true
		}
	}
	return false
}

// exec runs `git <sub> <args...>` after verifying sub is on the allow-list.
func (c *Client) exec(ctx context.Context, sub string, args ...string) (string, error) {
	if _, ok := allowedSubcommands[sub]; !ok {
		return "", fmt.Errorf("git subcommand %q is not allowed by doit", sub)
	}
	full := append([]string{sub}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = c.repo
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s: %w (%s)", sub, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
