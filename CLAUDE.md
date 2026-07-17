# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## VCS

Use **jujutsu** (`jj`), not `git`. Never run `git` commands directly.

## Build & Test Commands

```bash
mise run build       # go build → dist/go-outside
mise run test        # gotestsum, race detector, coverage → bin/coverage.out
mise run lint        # golangci-lint --fix → bin/golangci-lint.html
mise run generate    # go generate ./... (stringer for iota enums)
mise run mock        # regenerate mocks via mockery
mise run dev         # generate + lint + test + snapshot + mock
mise run snapshot    # goreleaser build (single target, no release)
mise run cover       # coverage HTML → bin/coverage.html
mise run mod-tidy    # go mod tidy + gomod2nix generate

# Run a single test package
gotestsum -- -race ./internal/...

# Run one test by name
gotestsum -- -run TestFoo ./...
```

Always run `mise run lint` and `mise run test` before committing.

## Task Completion Checklist

1. Fix all warnings (`mise run lint`, `mise run test`, compiler)
2. Run `/audit-docs` after features or fixes

## Architecture

<!-- TODO: describe the high-level architecture (e.g. event sourcing, layered, hexagonal) -->

## Repository Layout

```
<!-- TODO: fill in directory tree with descriptions -->
```

## Key Patterns

<!-- TODO: document constructor patterns, output conventions, testing approach, etc. -->

## Code Conventions

- Go 1.26.3, no CGo
- **Never redeclare a persistent flag as a local flag.** Persistent flags (`--quiet`/`-q`, `--json`, `--config`) are defined on the root command and flow into the config via context.
- `testify/assert` for non-fatal assertions, `testify/require` for preconditions
- Wrap errors: `fmt.Errorf("context: %w", err)`
- Timestamps: RFC3339 with UTC `Z` suffix
- No type assertions / `any` casts — prefer typed interfaces
- No magic numbers — use stdlib constants
- UI: silence is success; verbose output only with `-v`/`--verbose`; `--json` on all commands

### Fakes vs. mocks

**Prefer hand-rolled fakes for internal interfaces.** Reserve `mockery`-generated mocks for external dependencies only.

## Shell Tooling Preferences

- `fd -H` over `find`, `rg` over `grep`, `sd` over `sed`, `jq` for JSON
- No `&&` between shell commands — run as separate tool calls
- No `git` commands — use `jj` equivalents

## Agent skills

### Issue tracker

Issues and PRDs live in GitHub Issues (`asphaltbuffet/go-outside`), managed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles map to identically-named GitHub labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
