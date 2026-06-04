# Canopy Development Guide

This file provides context and guidelines for working on the Canopy project.

## Project Overview

Canopy is a local Go CLI that highlights uncovered lines in a diff. It parses Go
coverage files from a directory, intersects them with a git diff, and prints the
uncovered lines.

Three output formats are supported:

- **Text** — human-readable output for the terminal
- **Markdown** — for PR comments or documentation
- **GitHubAnnotations** — `::notice file=…,line=…::…` workflow commands that GitHub
  Actions renders as PR annotations. This is the entire "posting" mechanism — no
  GitHub API client involved.

Three diff sources are supported:

- Local working tree (default) — `git diff`
- A single commit — `--commit <sha>`
- A base ref against HEAD or another ref — `--base <ref>` (optionally `--commit <ref>`)

## Code Organization

```
cmd/canopy/        # CLI entry point (cobra)
internal/
├── coverage/      # Coverage parsing, merging (gocovmerge algorithm), analysis
├── diff/          # DiffSource interface + local/commit/base implementations
├── format/        # Text, Markdown, GitHubAnnotations formatters
├── github/        # Annotation data structures (Annotation, LineRange, grouping helpers)
└── local/         # Runner: orchestrates diff → parse → merge → analyze → format
```

`internal/github` only holds data structures and grouping helpers used by the
formatters and `coverage.GenerateAnnotations` — there is no GitHub API client.

## Development Guidelines

### Testing
- Use table-driven tests with clear test case names.
- Plan tests alongside implementation.
- Run `make test` after every change.

### Coverage Merging
- Use the gocovmerge algorithm (proven, well-tested).
- Leverage `golang.org/x/tools/cover` for parsing.
- Merge at block level for accuracy.

### Code Style
- Follow standard Go conventions.
- Use structured logging where logs are needed.
- Use context for cancellation and timeouts.

## Common Commands

```bash
make build              # Build the canopy binary
make test               # Run tests (-race -short)
make test-coverage      # Run tests with coverage report
make coverage-html      # Generate and open HTML coverage report
make lint               # go fmt + go vet
make install            # Install canopy to $GOPATH/bin
make clean              # Remove build artifacts
```

## Dependencies

Key packages:
- `github.com/spf13/cobra` — CLI framework
- `golang.org/x/tools/cover` — Coverage parsing
- `github.com/stretchr/testify` — Test assertions

## When Working on This Project

1. **Before modifying code**: Read existing implementation to understand patterns.
2. **When adding features**: Follow existing interface patterns (e.g., `DiffSource`,
   `Formatter`).
3. **When adding tests**: Aim for table-driven tests with clear test case names.
4. **Test edge cases thoroughly**: empty files, missing data, malformed input.
