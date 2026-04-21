package errs

import "fmt"

type ExitCode int

const (
	ExitSuccess           ExitCode = 0
	ExitGeneric           ExitCode = 1
	ExitAuth              ExitCode = 2
	ExitUserInput         ExitCode = 3
	ExitDependencyConflict ExitCode = 4
	ExitUpstream          ExitCode = 5
)

type CLIError struct {
	Code ExitCode
	Err  error
	// Silent indicates the command has already written its formatted error to
	// stderr. The CLI entrypoint must skip the default "Error: <msg>" line
	// and only apply the exit code.
	Silent bool
}

func (e *CLIError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

func New(code ExitCode, format string, args ...any) *CLIError {
	return &CLIError{
		Code: code,
		Err:  fmt.Errorf(format, args...),
	}
}

func Wrap(code ExitCode, err error) *CLIError {
	if err == nil {
		return nil
	}
	if existing, ok := err.(*CLIError); ok {
		return existing
	}
	return &CLIError{Code: code, Err: err}
}

// SilentExit returns a CLIError that carries only an exit code; the caller
// has already written the user-facing message to stderr.
func SilentExit(code ExitCode) *CLIError {
	return &CLIError{Code: code, Silent: true}
}

// IsSilent reports whether main should skip the default error line.
func IsSilent(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*CLIError); ok {
		return e.Silent
	}
	return false
}

func CodeOf(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}
	if e, ok := err.(*CLIError); ok {
		return e.Code
	}
	return ExitGeneric
}
