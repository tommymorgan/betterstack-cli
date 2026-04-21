package bodyfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

const MaxBytes = 10 * 1024 * 1024 // 10 MB

// Read reads the body-file argument (path or "-" for stdin), validates that
// it is well-formed JSON and within the size cap, and returns the raw bytes
// so callers can forward them verbatim.
func Read(flag string, stdin io.Reader) ([]byte, error) {
	if flag == "" {
		return nil, errs.New(errs.ExitUserInput, "--body-file path is empty")
	}

	data, err := readSource(flag, stdin)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errs.New(errs.ExitUserInput,
			"--body-file content is empty; an empty body is not a valid request")
	}

	if !json.Valid(data) {
		return nil, errs.New(errs.ExitUserInput,
			"--body-file content is not valid JSON")
	}

	return data, nil
}

func readSource(flag string, stdin io.Reader) ([]byte, error) {
	if flag == "-" {
		return readCapped(stdin, "stdin exceeds the 10MB --body-file cap")
	}

	info, err := os.Stat(flag)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errs.New(errs.ExitUserInput, "--body-file path %q does not exist", flag)
	}
	if err != nil {
		return nil, errs.New(errs.ExitUserInput, "--body-file path %q is not accessible: %s", flag, err.Error())
	}
	if info.IsDir() {
		return nil, errs.New(errs.ExitUserInput, "--body-file path %q is a directory", flag)
	}
	if info.Size() > MaxBytes {
		return nil, errs.New(errs.ExitUserInput,
			"--body-file %q is %d bytes; exceeds the 10MB cap", flag, info.Size())
	}

	f, err := os.Open(flag)
	if err != nil {
		return nil, errs.New(errs.ExitUserInput, "--body-file %q could not be read: %s", flag, err.Error())
	}
	defer f.Close()

	return readCapped(f, fmt.Sprintf("%q exceeds the 10MB --body-file cap", flag))
}

func readCapped(r io.Reader, overflowMsg string) ([]byte, error) {
	limited := io.LimitReader(r, MaxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, errs.New(errs.ExitUserInput, "failed to read --body-file: %s", err.Error())
	}
	if len(data) > MaxBytes {
		return nil, errs.New(errs.ExitUserInput, "%s", overflowMsg)
	}
	return data, nil
}
