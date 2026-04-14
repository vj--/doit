# doit

[![Release](https://img.shields.io/github/v/release/vj--/doit?label=release)](https://github.com/vj--/doit/releases/latest)
[![CI](https://github.com/vj--/doit/actions/workflows/ci.yml/badge.svg)](https://github.com/vj--/doit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A local-first, Kanban-style todo app that lives in your **terminal** and is backed by a **git repository** instead of a database. Your tasks are a single markdown file in a repo you control — the app renders a keyboard-driven Kanban TUI, and commits every change locally. You push to GitHub (or any remote) whenever you want.

> **Note:** This project is **AI-generated**. It was designed and implemented with [Claude Code](https://claude.com/claude-code) (Anthropic's CLI coding assistant) in collaboration with the repository owner. Treat it accordingly — review the code before you trust it with anything important.

## Why

- **Your data, your repo.** Tasks are plain markdown. Readable in any editor, diffable, grep-able, forkable.
- **History for free.** Every edit is a git commit. `git log` is your audit trail; `git checkout` is your undo.
- **Multi-device sync without a server.** Push/pull with any git remote.
- **Terminal-native.** Works over SSH. No browser, no local web server, no ports.
- **Single binary.** No database, no Docker, no runtime dependencies.

## How it works

1. Point the CLI at a git-managed folder containing a tasks markdown file.
2. The app opens a Kanban board in your terminal.
3. Move cards with the keyboard, edit tasks, add new ones — every change is written back to the markdown file and committed locally.
4. You run `git push` when you want to sync to a remote. The app never pushes for you.

## Install

### Download a release (recommended)
Grab the binary for your platform from the [Releases page](https://github.com/vj/doit/releases), extract it, and put it on your `PATH`.

Supported platforms: macOS (Intel + Apple Silicon), Linux (amd64/arm64), Windows (amd64).

### Go install
```sh
go install github.com/vj/doit@latest
```

> Building from source? See [`DEVELOPMENT.md`](./DEVELOPMENT.md).

## Quickstart (from scratch)

If you've never used git before, this walks you from zero to a running board. Skip the steps you've already done.

### 1. Install git
- **macOS**: `xcode-select --install` (or `brew install git`).
- **Linux**: `sudo apt install git` / `sudo dnf install git` / your distro's equivalent.
- **Windows**: download from [git-scm.com](https://git-scm.com/download/win).

Verify:
```sh
git --version
```

### 2. Tell git who you are (one-time, global)
```sh
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

### 3. Create a repo to hold your tasks
Anywhere you like:
```sh
mkdir ~/my-tasks
cd ~/my-tasks
git init
```
That's it — you now have a local git repository. No server, no GitHub required yet.

### 4. Run doit against it
```sh
doit --repo ~/my-tasks
```
On first launch the TUI prompts `Create board.md? [Y/n]`. Press `Y`. You'll see three columns: **Todo**, **In Progress**, **Done**. Press `n` to add a card, `h/j/k/l` to navigate, `H/L` to move a card between columns, `?` for help, `q` to quit. Every action is committed locally.

Check it worked:
```sh
cd ~/my-tasks
git log --oneline
```
You should see one commit per action.

### 5. (Optional) Push to GitHub for backup / sync
Create an empty repo on [github.com/new](https://github.com/new) — **do not** add a README, license, or `.gitignore` there (that would conflict with your existing local commits). Then:
```sh
cd ~/my-tasks
git remote add origin https://github.com/YOUR-USERNAME/my-tasks.git
git branch -M main
git push -u origin main
```
From now on, run `git push` whenever you want to back up your work. **doit never pushes for you.**

## Usage

```sh
doit --repo ~/my-tasks-repo
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--repo <path>` | `.` | Path to the git-managed folder holding the tasks file. |
| `--file <name>` | `board.md` | Markdown file inside the repo. |
| `--no-commit` | off | Edit the file without committing. |
| `--theme <light\|dark>` | auto | Force UI theme. Useful when running inside `tmux`, where background auto-detection often fails. |
| `--config <path>` | *(platform default)* | Path to a config file. See [Config file](#config-file). |
| `--version` | | Print version and exit. |

### Theme

doit auto-detects your terminal's light/dark background. Inside `tmux` (and some remote shells) that detection fails, and colors can come out washed-out or unreadable. Force it with either:

```sh
doit --theme light   # or: dark
DOIT_THEME=light doit
```

Precedence: `--theme` flag > `DOIT_THEME` env var > `theme` in config file > auto-detect.

### Config file

Optional. Set defaults so you don't have to pass the same flags every time. Location by platform (created by hand — doit does not auto-create it):

| OS | Default path |
|---|---|
| macOS | `~/Library/Application Support/doit/config.toml` |
| Linux | `$XDG_CONFIG_HOME/doit/config.toml` (typically `~/.config/doit/config.toml`) |
| Windows | `%AppData%\doit\config.toml` |

Override the location with `--config <path>`.

Format is simple `key = value` (one per line, `#` for comments — TOML-compatible subset):

```toml
# ~/.config/doit/config.toml
repo      = "~/my-tasks"
file      = "board.md"
theme     = "light"
no_commit = false
```

Supported keys: `repo`, `file`, `theme`, `no_commit`. A leading `~/` in `repo` is expanded to your home directory.

Precedence for every setting: **CLI flag > config file > built-in default**.

### Keybindings

| Keys | Action |
|---|---|
| `h` / `l` (or `←` / `→`) | Focus previous / next column |
| `j` / `k` (or `↓` / `↑`) | Focus previous / next card |
| `H` / `L` | Move focused card to column left / right |
| `J` / `K` | Reorder focused card within column |
| `n` | New card |
| `e` / `Enter` | Edit focused card |
| `d` | Delete focused card (with confirm) |
| `/` | Filter cards |
| `?` | Toggle help |
| `q` / `Ctrl+C` | Quit (flushes pending commit) |

## Stack

Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Bubbles](https://github.com/charmbracelet/bubbles) + [goldmark](https://github.com/yuin/goldmark). Shells out to the system `git` binary. Single static binary.

## Status

Early / pre-release. Design and scaffolding are in progress. APIs, flags, and the markdown schema may change before 1.0.

## Contributing

Issues and PRs welcome once the first release is out. Since the codebase is AI-generated, please keep PR descriptions explicit about whether your changes were written by hand, with AI assistance, or both. See [`DEVELOPMENT.md`](./DEVELOPMENT.md) for build and dev setup.

## License

[MIT](./LICENSE).
