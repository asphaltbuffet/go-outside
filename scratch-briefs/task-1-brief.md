# Task 1 Brief — resty dependency + exit-code sentinel

These are your requirements. Use the exact values, signatures, and test cases below verbatim.

## Global Constraints (bind every task)

- Go 1.26.3, no CGo.
- `testify/assert` for non-fatal assertions, `testify/require` for preconditions.
- Wrap errors: `fmt.Errorf("context: %w", err)`.
- No type assertions / `any` casts outside the established context-key pattern — prefer typed interfaces.
- No magic numbers — name constants.
- Prefer hand-rolled fakes for internal interfaces; reserve mockery for external deps.
- **VCS is `jj`, not `git`.** For the commit step, use `jj commit -m "..."` (NOT `git commit`). Do not run any `git` command.
- Exit codes per ADR-0002: `ok`=0, `nope`=1, error=2, in every output mode. This task builds the mechanism that carries those codes.

## Task 1: resty dependency + exit-code sentinel

Adds the HTTP dependency and the typed error that carries the process exit code. Foundation for every later task.

**Files:**
- Modify: `go.mod`, `go.sum` (via `go get`)
- Create: `internal/cli/exit.go`
- Test: `internal/cli/exit_test.go`

**Interfaces produced (later tasks depend on these exact names):**
- `cli.ExitError` — `type ExitError struct { Code int; Err error }`; methods `Error() string`, `Unwrap() error`.
- `cli.NewExitError(code int, err error) *ExitError`
- `cli.CodeFor(err error) int` — returns the `Code` of the first `*ExitError` in the chain, `0` if `err == nil`, else `2`.

### Step 1: Add resty

Run:
```bash
go get github.com/go-resty/resty/v2@v2.16.5
```
Expected: `go.mod` gains `github.com/go-resty/resty/v2 v2.16.5`.

### Step 2: Write the failing test

Create `internal/cli/exit_test.go`:
```go
package cli_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/cli"
)

func TestCodeFor(t *testing.T) {
	assert.Equal(t, 0, cli.CodeFor(nil))

	plain := errors.New("boom")
	assert.Equal(t, 2, cli.CodeFor(plain))

	nope := cli.NewExitError(1, errors.New("nope"))
	assert.Equal(t, 1, cli.CodeFor(nope))

	wrapped := errors.New("wrap: " + nope.Error())
	_ = wrapped
	assert.ErrorIs(t, nope, nope.Unwrap())
}

func TestExitErrorWrap(t *testing.T) {
	inner := errors.New("inner")
	ee := cli.NewExitError(2, inner)
	assert.Equal(t, "inner", ee.Error())
	assert.ErrorIs(t, ee, inner)
}
```

### Step 3: Run test to verify it fails

Run: `gotestsum -- -run TestCodeFor ./internal/cli/`
Expected: FAIL — `undefined: cli.NewExitError` / `cli.CodeFor`.

### Step 4: Write minimal implementation

Create `internal/cli/exit.go`:
```go
package cli

import "errors"

// ExitError carries a process exit code alongside an error, so the command
// layer can map a Verdict (nope=1) or a failure (2) to os.Exit. See ADR-0002.
type ExitError struct {
	Code int
	Err  error
}

// NewExitError builds an ExitError with the given exit code and cause.
func NewExitError(code int, err error) *ExitError {
	return &ExitError{Code: code, Err: err}
}

func (e *ExitError) Error() string { return e.Err.Error() }

func (e *ExitError) Unwrap() error { return e.Err }

// CodeFor returns the exit code for err: 0 for nil, the ExitError's Code if
// one is present in the chain, otherwise 2 (an unclassified failure).
func CodeFor(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return 2
}
```

### Step 5: Run test to verify it passes

Run: `gotestsum -- -run 'TestCodeFor|TestExitErrorWrap' ./internal/cli/`
Expected: PASS.

### Step 6: Commit

```bash
jj commit -m "feat(cli): add ExitError sentinel and resty dependency"
```

Note: `internal/cli/` already contains `output.go` and `flags.go` in package `cli`. Your new `exit.go` joins that package — do not create a new package or a `doc.go`.
