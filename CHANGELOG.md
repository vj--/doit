# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.2] - 2026-04-14

Re-releases 0.2.1 so the `CHANGELOG.md` bundled in the archives is up to date. No code or binary changes.

## [0.2.1] - 2026-04-14

### Added
- macOS release binaries are now signed and notarized with a Developer ID, so `doit` runs on macOS without Gatekeeper's "unidentified developer" prompt.

## [0.2.0] - 2026-04-14

### Added
- Charm-inspired minimalist TUI refresh: spinner, help bar, glamour-rendered task detail pane, tag-colored label pills, and a priority stripe on focused cards.
- Markdown rendering in the task detail pane (code fences, lists, GFM checklists).
- Time-based hiding of Done tasks: `--hide-done-after <days>` flag (default 5) and `hide_done_after_days` config key. Press `a` to toggle visibility; columns show a `+N hidden · press a` footer.
- Edit labels directly from the edit modal — Tab cycles title → body → labels.
- README badges (release, CI, MIT license) and board screenshots.

### Fixed
- Task bodies with blank-line paragraph breaks, GFM task checklists, or fenced code blocks no longer produce ghost cards on reload.
- Light-mode terminal detection: glamour now uses the same theme signal as the rest of the UI, fixing an unreadable detail pane in `tmux` and similar environments.

## [0.1.0] - 2026-04-13

### Added
- Initial release: Kanban TUI backed by a markdown file in a git repo.
- Keyboard navigation (`h/j/k/l`), column move (`H/L`), within-column reorder (`J/K`), create/edit/delete, substring filter.
- Per-action local git commits with descriptive messages; never pushes.
- CLI flags: `--repo`, `--file`, `--no-commit`, `--theme`, `--config`, `--version`.
- TOML-compatible config file with platform-specific default paths.
- Cross-platform release archives (macOS amd64/arm64, Linux amd64/arm64, Windows amd64) via GoReleaser.
