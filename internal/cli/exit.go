package cli

import "errors"

const (
	// ExitCodeOK is the process exit code for an ok Verdict.
	ExitCodeOK = 0
	// ExitCodeNope is the process exit code for a nope Verdict.
	ExitCodeNope = 1
	// ExitCodeError is the process exit code for a failure to assess.
	ExitCodeError = 2
)

// ExitError carries a process exit code alongside an error, so the command
// layer can map a Verdict (nope=1) or a failure (2) to [os.Exit]. See ADR-0002.
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
		return ExitCodeOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitCodeError
}
