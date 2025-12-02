# Canopy

Canopy is a Go code coverage analysis tool that allows to highlight uncovered lines in diff - based on local working directory, specific commit or diff between HEAD and base commit

## Features

- **Local Coverage Analysis**: Analyze coverage files against your git diff
- **Multiple Diff Modes**: Analyze uncommitted changes, commits, or PR branches
- **Flexible Output**: Text, Markdown, or GitHub Annotations format
- **GitHub Integration**: Automated PR check runs and comments (via webhook handlers)
- **Simple CLI**: Easy to use with sensible defaults

## Quickstart

### Installation

Install Canopy using `go install`:

```bash
go install github.com/oleg-kozlyuk-grafana/go-canopy/cmd/canopy@latest
```

This will install the `canopy` binary to your `$GOPATH/bin` directory (typically `~/go/bin`).

Make sure `$GOPATH/bin` is in your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Basic Usage

1. **Generate coverage data** for your project:

```bash
go test -coverprofile=.coverage/coverage.out ./...
```

2. **Analyze your local changes**:

```bash
canopy --coverage .coverage
```

This will:
- Merge all coverage files in `.coverage/` directory
- Compare against your current git diff
- Show uncovered lines in your changes

### Example Output

```
Found 1 coverage file(s) to merge
Uncovered lines in diff:

internal/math/subtract.go
  Lines: 4-6, 9-17

Summary: 12 uncovered lines out of 33 added lines (63.6% coverage)
```

## Usage Modes

### Analyze Uncommitted Changes (Default)

Analyze coverage for your current working directory changes:

```bash
canopy --coverage .coverage
```

### Analyze Against Base Branch

Analyze coverage for all changes between a base ref and HEAD (useful for PRs):

```bash
canopy --coverage .coverage --base main
```

This compares your current branch against `main` to show uncovered lines in your PR.

### Analyze Specific Commit

Analyze coverage for a specific commit:

```bash
canopy --coverage .coverage --commit abc123
```

## Output Formats

### Text (Default)

Human-readable output for terminal:

```bash
canopy --coverage .coverage --format Text
```

### Markdown

Markdown-formatted output for documentation or PR comments:

```bash
canopy --coverage .coverage --format Markdown
```

### GitHub Annotations

GitHub Actions annotation format for CI integration:

```bash
canopy --coverage .coverage --format GitHubAnnotations
```

## Configuration

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--coverage` | `.coverage` | Directory containing coverage files |
| `--format` | `Text` | Output format (Text, Markdown, GitHubAnnotations) |
| `--base` | - | Analyze diff against base ref (e.g., `main`) |
| `--commit` | - | Analyze diff for specific commit SHA |

### Coverage File Location

By default, Canopy looks for coverage files in `.coverage/` directory. You can specify a different location:

```bash
canopy --coverage /path/to/coverage
```

Canopy will merge all `.out` files found in the specified directory.

## GitHub Integration

Canopy can also run as a GitHub webhook handler to automatically:
- Process coverage from GitHub Actions workflows
- Create check runs on PRs
- Post coverage comments with before/after comparison
- Fail checks if coverage decreases

See [CLAUDE.md](CLAUDE.md) for development setup and [SPEC.md](SPEC.md) for architecture details.

## Common Workflows

### Local Development

```bash
# Run tests with coverage
go test -coverprofile=.coverage/coverage.out ./...

# Check coverage for your changes
canopy --coverage .coverage
```

### Pre-Commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
go test -coverprofile=.coverage/coverage.out ./... && \
canopy --coverage .coverage --format Text
```

### CI Integration (GitHub Actions)

```yaml
- name: Run tests with coverage
  run: go test -coverprofile=.coverage/coverage.out ./...

- name: Analyze coverage
  run: |
    go install github.com/oleg-kozlyuk-grafana/go-canopy/cmd/canopy@latest
    canopy --coverage .coverage --base ${{ github.base_ref }} --format GitHubAnnotations
```

## Version Information

Check your installed version:

```bash
canopy version
```

## Contributing

For development setup and contribution guidelines, see:
- [CLAUDE.md](CLAUDE.md) - Development guide and architecture
- [PLAN.md](PLAN.md) - Implementation plan and progress
- [SPEC.md](SPEC.md) - Technical specification

### Development Setup

```bash
# Clone the repository
git clone https://github.com/oleg-kozlyuk-grafana/go-canopy.git
cd go-canopy

# Build the project
make build

# Run tests
make test

# Run tests with coverage
make test-coverage
```

## License

See [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- See [CLAUDE.md](CLAUDE.md) for architecture details
- Check [SPEC.md](SPEC.md) for feature specifications
