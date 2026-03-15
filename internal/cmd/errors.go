package cmd

import "fmt"

// ExitError wraps an error with a specific process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ExitErrorf creates a new ExitError with a formatted message.
func ExitErrorf(code int, format string, args ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, args...)}
}
