package git

import (
	"context"
	"testing"
)

func TestDisallowedSubcommandsAreNotAllowed(t *testing.T) {
	forbidden := []string{"push", "pull", "fetch", "merge", "rebase", "reset", "checkout", "stash", "tag", "remote"}
	for _, sub := range forbidden {
		if _, ok := allowedSubcommands[sub]; ok {
			t.Errorf("%q must not be on the allow-list", sub)
		}
	}
}

func TestExecRejectsUnknownSubcommand(t *testing.T) {
	c := New(".")
	_, err := c.exec(context.Background(), "push", "origin", "main")
	if err == nil {
		t.Fatal("expected error for disallowed subcommand")
	}
}
