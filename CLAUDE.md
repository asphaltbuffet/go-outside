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

Layered CLI. `cmd/` (Cobra) is a thin shell that loads config, builds an `App`, and formats output; all domain logic lives behind the `App` seam in `internal/app`. The core flow is **Location → Conditions → Verdict** (see `CONTEXT.md`): resolve a Location (auto-detect or geocode an override), fetch Conditions from keyless Open-Meteo providers, compute a discomfort **Score** by summing per-Factor penalties, and compare it to a **Threshold** to produce an `ok`/`nope` **Verdict**. Providers are small interfaces with resty-backed real implementations and hand-rolled fakes, injected into the `App` — so the domain is testable without network. The Verdict is also the process exit status: `ok`=0, `nope`=1, error=2 (ADR-0002).

## Repository Layout

```
main.go                     entrypoint; maps cli.CodeFor(err) → os.Exit (ADR-0002)
cmd/
  root.go                   root command: flags, RunE, output modes, appFactory seam
  version.go                version subcommand
internal/
  app/        App           domain orchestrator; App.Assess(ctx, override) → Assessment
  score/      score.go      pure scoring: Factor penalty curves, Score, Verdict (ADR-0004)
  weather/    Provider      Open-Meteo weather + AQI → weather.Report (+ FakeProvider)
  geo/        Locator/Geocoder  IP auto-detect + postal-code geocode → geo.Location (+ fakes)
  config/     Config        config load + Threshold knob (the only user-tunable value)
  cli/        OutputWriter, ExitError  output formatting + exit-code sentinel
  apptesting/ OpenApp       builds an App wired with fakes for tests
  version/                  build-version info
```

## Key Patterns

- **Provider interfaces + hand-rolled fakes.** `geo.Locator`, `geo.Geocoder`, `weather.Provider` are injected into `app.New`. Real impls take a `*resty.Client` (nil → production base URLs; a BaseURL-set client routes to a test `httptest` server). Tests use the `Fake*` structs, never the network.
- **Errors are never a Verdict.** An inability to determine Location or Conditions is returned as a wrapped error (exit 2), never collapsed into `nope` (ADR-0002).
- **Scoring is pure.** `score.Assess(report, threshold)` has no I/O; penalty curves are named constants calibrated in ADR-0004.
- **Exit codes via a typed sentinel.** `cli.ExitError{Code}` carries the exit code through `RunE → Execute → main`; `cli.CodeFor(err)` maps any error to its code. Cobra's `SilenceErrors`/`SilenceUsage` are on, so `main.go` prints genuine (exit-2) failures itself.
- **Output modes.** Default prints the bare Verdict; `-v`/`--verbose` prints `nope (999) [22901]`; `--json` always emits the full object. `w.Success` is suppressed under `--quiet`.

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
