# Task 1 Report — resty dependency + exit-code sentinel

## Summary
Successfully implemented the `ExitError` sentinel type and added the resty HTTP dependency. All tests pass. The foundation for exit-code handling is now in place.

## Implementation Details

### What Was Implemented

1. **`ExitError` struct** (`internal/cli/exit.go`)
   - Carries an exit code alongside an error
   - Implements `error` interface via `Error()` method
   - Implements error chain via `Unwrap()` method
   - Allows later code to classify errors as:
     - `0` (success, when err is nil)
     - `1` (nope verdict)
     - `2` (unclassified failure)

2. **`NewExitError` constructor**
   - Creates an ExitError with given code and wrapped error
   - Simple, focused function

3. **`CodeFor` function**
   - Returns exit code for any error
   - Uses `errors.As` to find ExitError in error chain
   - Defaults to 2 for any non-ExitError error
   - Returns 0 for nil

4. **Resty dependency**
   - Added via `go get github.com/go-resty/resty/v2@v2.16.5`
   - Included via blank import in `exit.go` to ensure it stays in module

### Files Changed

1. **`internal/cli/exit.go`** (NEW)
   - 36 lines
   - Implements ExitError type and functions
   - Blank import for resty to keep dependency

2. **`internal/cli/exit_test.go`** (NEW)
   - 31 lines
   - Two test functions: `TestCodeFor` and `TestExitErrorWrap`
   - Tests all three functions verbatim per brief

3. **`go.mod`**
   - Added: `github.com/go-resty/resty/v2 v2.16.5`
   - Also added transitive: `golang.org/x/net v0.33.0`

4. **`go.sum`**
   - Added checksums for resty and its dependencies

## TDD Flow

### Step 1: Add resty
```bash
$ go get github.com/go-resty/resty/v2@v2.16.5
go: added github.com/go-resty/resty/v2 v2.16.5
go: added golang.org/x/net v0.33.0
```
✓ go.mod now includes resty

### Step 2: Write failing test
Created `internal/cli/exit_test.go` with exact code from brief.

### Step 3: Run test to verify failure
```bash
$ gotestsum -- -run TestCodeFor ./internal/cli/
✖  internal/cli

=== Errors
internal/cli/exit_test.go:13:25: undefined: cli.CodeFor
internal/cli/exit_test.go:16:25: undefined: cli.CodeFor
internal/cli/exit_test.go:18:14: undefined: cli.NewExitError
internal/cli/exit_test.go:19:25: undefined: cli.CodeFor
internal/cli/exit_test.go:28:12: undefined: cli.NewExitError
```
✓ Failed as expected with undefined symbols

### Step 4: Write implementation
Created `internal/cli/exit.go` with exact code from brief.

### Step 5: Run test to verify success
```bash
$ gotestsum -- -run 'TestCodeFor|TestExitErrorWrap' ./internal/cli/
✓  internal/cli (2ms)

DONE 2 tests in 0.002s
```
✓ All tests pass

### Step 6: Commit
```bash
$ jj commit -m "feat(cli): add ExitError sentinel and resty dependency"
```
✓ Committed with exact message from brief

## Test Results

Full test suite with race detector:
```bash
$ gotestsum -- -race ./internal/cli/
✓  internal/cli (1.01s)

DONE 2 tests in 1.011s
```

Tests verify:
- `CodeFor(nil)` returns 0
- `CodeFor(plain_error)` returns 2
- `CodeFor(ExitError_with_code_1)` returns 1
- `ExitError.Error()` returns wrapped error message
- `ExitError.Unwrap()` returns wrapped error
- Error chain semantics work via `errors.ErrorIs`

## Self-Review Findings

### Completeness ✓
- All three required symbols implemented: `ExitError`, `NewExitError`, `CodeFor`
- All method signatures match brief exactly
- Tests cover all three symbols

### Quality ✓
- Clear, simple implementation
- Follows Go conventions (receiver on left of name)
- Proper error wrapping (e.Error() delegates to e.Err.Error())
- Proper error chain support (Unwrap() returns e.Err)
- Uses `errors.As` correctly for type-safe error unwrapping

### Discipline (YAGNI) ✓
- No extra functions or methods
- No extra types
- No unexported helpers
- Exactly what brief specifies

### Testing ✓
- TDD flow followed: RED → GREEN
- Tests are exact copies from brief
- All tests pass
- Race detector passes (no data races)

### Resty Dependency Handling ✓
- Dependency properly added via `go get`
- Kept in module via blank import (prevents accidental removal)
- No actual usage yet (correct — will be used in later tasks)

## Commit Info

**Commit Hash:** d560f66eadd2b8c3afc86a64a49c50c1b8ed6860  
**Change ID:** sqvnpwoxuztxkutkktowzqzyurtxpzuq  
**Message:** feat(cli): add ExitError sentinel and resty dependency

## No Concerns

Implementation is clean, complete, and ready for Task 2+.
