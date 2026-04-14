package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vj--/doit/internal/config"
	"github.com/vj--/doit/internal/git"
	"github.com/vj--/doit/internal/store"
	"github.com/vj--/doit/internal/tui"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("doit", flag.ContinueOnError)
	repo := fs.String("repo", ".", "path to the git-managed folder holding the tasks file")
	file := fs.String("file", "board.md", "markdown file inside the repo")
	noCommit := fs.Bool("no-commit", false, "edit the file without committing")
	configPath := fs.String("config", "", "path to config file (default: platform user config dir)")
	theme := fs.String("theme", "", "force UI theme: light|dark (overrides auto-detect; useful inside tmux)")
	hideDoneAfter := fs.Int("hide-done-after", 5, "hide tasks in Done older than N days (0 = never hide)")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		fmt.Printf("doit %s (%s, %s)\n", version, shortCommit(commit), date)
		return nil
	}
	if fs.NArg() > 0 && *repo == "." {
		*repo = fs.Arg(0)
	}

	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })

	cfgPath := *configPath
	if cfgPath == "" {
		p, err := config.DefaultPath()
		if err == nil {
			cfgPath = p
		}
	}
	fileCfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", cfgPath, err)
	}
	if !setFlags["repo"] && fileCfg.Repo != "" {
		*repo = fileCfg.Repo
	}
	if !setFlags["file"] && fileCfg.File != "" {
		*file = fileCfg.File
	}
	if !setFlags["no-commit"] && fileCfg.NoCommit != nil {
		*noCommit = *fileCfg.NoCommit
	}
	if !setFlags["hide-done-after"] && fileCfg.HideDoneAfterDays != nil {
		*hideDoneAfter = *fileCfg.HideDoneAfterDays
	}
	// Theme precedence: --theme flag > DOIT_THEME env > config file.
	// (env was already applied in tui.init(); flag/config take precedence over it.)
	switch {
	case *theme != "":
		tui.SetTheme(*theme)
	case os.Getenv("DOIT_THEME") == "" && fileCfg.Theme != "":
		tui.SetTheme(fileCfg.Theme)
	}

	absRepo, err := filepath.Abs(*repo)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	cfg := config.Config{
		Repo:              absRepo,
		File:              *file,
		NoCommit:          *noCommit,
		HideDoneAfterDays: *hideDoneAfter,
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	if !stdoutIsTTY() {
		return errors.New("doit is an interactive TUI and requires a terminal on stdout")
	}

	gitClient := git.New(cfg.Repo)
	s := store.New(cfg.FilePath(), gitClient, cfg.NoCommit)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	model, err := tui.NewModel(ctx, cfg, s)
	if err != nil {
		return err
	}

	prog := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)
	_, err = prog.Run()
	return err
}

func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	if c == "" {
		return "unknown"
	}
	return c
}
