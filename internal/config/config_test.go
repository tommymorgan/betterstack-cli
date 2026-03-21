package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveToken_EnvVarTakesPrecedence(t *testing.T) {
	t.Setenv(envVarName, "env-token")

	// Even if a config file exists, env var should win
	token, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("got %q, want %q", token, "env-token")
	}
}

func TestResolveToken_FallsBackToConfigFile(t *testing.T) {
	t.Setenv(envVarName, "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("api_token: file-token\n"), filePerms); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)

	token, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "file-token" {
		t.Errorf("got %q, want %q", token, "file-token")
	}
}

func TestResolveToken_ErrorWhenNoTokenConfigured(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv("HOME", t.TempDir())

	_, err := ResolveToken()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, envVarName) {
		t.Errorf("error should mention %s, got: %s", envVarName, msg)
	}
	if !strings.Contains(msg, "config.yaml") {
		t.Errorf("error should mention config.yaml, got: %s", msg)
	}
}

func TestResolveToken_WarnsOnOpenPermissions(t *testing.T) {
	t.Setenv(envVarName, "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Write with overly permissive mode
	if err := os.WriteFile(configPath, []byte("api_token: file-token\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	token, err := ResolveToken()

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "file-token" {
		t.Errorf("got %q, want %q", token, "file-token")
	}

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])
	if !strings.Contains(output, "Warning") {
		t.Errorf("expected warning on stderr, got: %s", output)
	}
}

func TestResolveToken_EnvVarWinsOverConfigFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("api_token: file-token\n"), filePerms); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)
	t.Setenv(envVarName, "env-token")

	token, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("got %q, want %q", token, "env-token")
	}
}

func TestResolveToken_EmptyTokenInConfigFileIsError(t *testing.T) {
	t.Setenv(envVarName, "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("api_token: \"\"\n"), filePerms); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)

	_, err := ResolveToken()
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
