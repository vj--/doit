# Development

Build, test, and contribute to **doit**. This project is AI-generated (built with Claude Code) and open to community contributions.

## Before you start

- For anything non-trivial, **open an issue first** to discuss the design. Save yourself a rewrite.
- Check existing issues and PRs to avoid duplicate work.
- Read [`CLAUDE.md`](./CLAUDE.md) — it describes the architecture and the constraints this project operates under (especially the **git policy**: the app only commits locally, never pushes).

## Development setup

Requirements: Go 1.22+ and `git` on your `PATH`.

### Clone and fetch dependencies
```sh
git clone https://github.com/vj--/doit.git
cd doit
go mod tidy
```

### Build
```sh
go build -o bin/doit ./cmd/doit
./bin/doit --version
```

Or run directly without producing a binary:
```sh
go run ./cmd/doit --repo ./tmp-data
```

### Throwaway repo for local development
doit needs a git repository to point at. Create one inside the project:
```sh
mkdir -p tmp-data && (cd tmp-data && git init && git config user.email "dev@example.com" && git config user.name "Dev")
```
`tmp-data/` is already gitignored.

### Tests and checks
```sh
go test ./...           # unit tests
go vet ./...            # static checks
go fmt ./...            # formatting
```

### Local release dry-run
Produces per-platform archives under `dist/` without publishing. Requires [goreleaser](https://goreleaser.com/install/).
```sh
goreleaser release --snapshot --clean
```

### Project layout
See [`CLAUDE.md`](./CLAUDE.md) § Architecture for the package layout and the
constraints the codebase is built around (especially the git allow-list in
`internal/git`).

## Pull requests

- Keep PRs focused. One logical change per PR.
- Include tests for new behavior where practical.
- Run `go fmt ./...`, `go vet ./...`, and `go test ./...` before submitting.
- Update `CLAUDE.md` and `README.md` if your change affects architecture, CLI flags, or the markdown schema.

### AI-assisted contributions

This codebase was originally generated with AI assistance, and AI-assisted PRs are welcome. Please **state clearly in the PR description** whether your changes were:
- written by hand,
- generated with AI assistance (and which tool),
- or a mix.

This isn't a gate — it's transparency for reviewers.

## Commit messages

Use short, conventional-style prefixes where they fit: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`. Release notes are generated from commit messages.

## Reporting bugs / requesting features

Use the issue templates under `.github/ISSUE_TEMPLATE/`.

## Code of conduct

By participating, you agree to abide by the [Code of Conduct](./CODE_OF_CONDUCT.md).
