package errs

import (
	"errors"
	"testing"
)

func TestCLIError_CarriesExitCodeWithMessage(t *testing.T) {
	err := New(ExitUserInput, "bad flag combo: %s", "x")
	if err.Code != ExitUserInput {
		t.Errorf("Code = %d, want %d", err.Code, ExitUserInput)
	}
	if err.Error() != "bad flag combo: x" {
		t.Errorf("Error() = %q, want %q", err.Error(), "bad flag combo: x")
	}
}

func TestCodeOf_ReturnsSuccessForNilError(t *testing.T) {
	if got := CodeOf(nil); got != ExitSuccess {
		t.Errorf("CodeOf(nil) = %d, want 0", got)
	}
}

func TestCodeOf_ReturnsGenericForPlainError(t *testing.T) {
	if got := CodeOf(errors.New("plain")); got != ExitGeneric {
		t.Errorf("CodeOf(plain) = %d, want 1", got)
	}
}

func TestCodeOf_ReturnsWrappedCode(t *testing.T) {
	err := New(ExitDependencyConflict, "deps exist")
	if got := CodeOf(err); got != ExitDependencyConflict {
		t.Errorf("CodeOf(CLIError{4}) = %d, want 4", got)
	}
}

func TestWrap_PreservesExistingCLIError(t *testing.T) {
	inner := New(ExitAuth, "bad token")
	wrapped := Wrap(ExitGeneric, inner)
	if wrapped.Code != ExitAuth {
		t.Errorf("Wrap should preserve inner code, got %d", wrapped.Code)
	}
}

func TestWrap_NilReturnsNil(t *testing.T) {
	if Wrap(ExitGeneric, nil) != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestCLIError_UnwrapReturnsInner(t *testing.T) {
	inner := errors.New("boom")
	err := Wrap(ExitUpstream, inner)
	if !errors.Is(err, inner) {
		t.Error("errors.Is should match inner error via Unwrap")
	}
}
