package bodyfile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

func TestRead_ReturnsFileContentsVerbatim(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	content := []byte(`{"name":"INF-3017","nested":{"a":[1,2,3]}}`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Read(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestRead_StdinDashReadsFromProvidedReader(t *testing.T) {
	content := []byte(`{"from":"stdin"}`)
	got, err := Read("-", bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestRead_ErrorsOnMissingPath(t *testing.T) {
	_, err := Read("/definitely/does/not/exist.json", nil)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if errs.CodeOf(err) != errs.ExitUserInput {
		t.Errorf("exit code = %d, want %d", errs.CodeOf(err), errs.ExitUserInput)
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should say 'does not exist', got: %s", err.Error())
	}
}

func TestRead_ErrorsOnInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	if err := os.WriteFile(path, []byte("not json { [ bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "valid JSON") {
		t.Errorf("error should mention valid JSON, got: %s", err.Error())
	}
}

func TestRead_ErrorsOnEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path, nil)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %s", err.Error())
	}
}

func TestRead_ErrorsOnEmptyStdin(t *testing.T) {
	_, err := Read("-", bytes.NewReader(nil))
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %s", err.Error())
	}
}

func TestRead_ErrorsOnOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	oversized := bytes.Repeat([]byte("a"), MaxBytes+10)
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path, nil)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "10MB") && !strings.Contains(err.Error(), "cap") {
		t.Errorf("error should mention size cap, got: %s", err.Error())
	}
}

func TestRead_ErrorsOnOversizedStdin(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), MaxBytes+10)
	_, err := Read("-", bytes.NewReader(oversized))
	if err == nil {
		t.Fatal("expected error for oversized stdin")
	}
	if !strings.Contains(err.Error(), "cap") {
		t.Errorf("error should mention cap, got: %s", err.Error())
	}
}

func TestRead_ErrorsOnDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := Read(dir, nil)
	if err == nil {
		t.Fatal("expected error for directory path")
	}
}
