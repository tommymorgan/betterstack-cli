package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
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

func TestResolveTelemetryToken_PrefersTelemetryEnvVar(t *testing.T) {
	t.Setenv(envVarName, "uptime-token")
	t.Setenv(telemetryEnvVarName, "telemetry-token")

	token, source, err := ResolveTelemetryToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "telemetry-token" {
		t.Errorf("token = %q, want telemetry-token", token)
	}
	if source != SourceTelemetryEnv {
		t.Errorf("source = %d, want SourceTelemetryEnv", source)
	}
}

func TestResolveTelemetryToken_FallsBackToUptimeEnvVar(t *testing.T) {
	t.Setenv(envVarName, "uptime-token")
	t.Setenv(telemetryEnvVarName, "")
	t.Setenv("HOME", t.TempDir())

	token, source, err := ResolveTelemetryToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "uptime-token" {
		t.Errorf("token = %q, want uptime-token", token)
	}
	if source != SourceUptimeFallback {
		t.Errorf("source = %d, want SourceUptimeFallback", source)
	}
}

func TestResolveTelemetryToken_UsesConfigFileTelemetryToken(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv(telemetryEnvVarName, "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "telemetry_api_token: from-file-telemetry\napi_token: from-file-uptime\n"
	if err := os.WriteFile(configPath, []byte(content), filePerms); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)

	token, source, err := ResolveTelemetryToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-file-telemetry" {
		t.Errorf("token = %q, want from-file-telemetry", token)
	}
	if source != SourceTelemetryFile {
		t.Errorf("source = %d, want SourceTelemetryFile", source)
	}
}

func TestResolveTelemetryToken_EnvVarWinsOverConfigFile(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv(telemetryEnvVarName, "from-env")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("telemetry_api_token: from-file\n"), filePerms); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)

	token, _, err := ResolveTelemetryToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-env" {
		t.Errorf("token = %q, want from-env", token)
	}
}

func TestResolveTelemetryToken_FallsBackToConfigFileUptimeToken(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv(telemetryEnvVarName, "")

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "betterstack", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("api_token: uptime-from-file\n"), filePerms); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir)

	token, source, err := ResolveTelemetryToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "uptime-from-file" {
		t.Errorf("token = %q, want uptime-from-file", token)
	}
	if source != SourceUptimeFallback {
		t.Errorf("source = %d, want SourceUptimeFallback", source)
	}
}

func TestResolveTelemetryToken_ErrorsWhenNothingConfigured(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv(telemetryEnvVarName, "")
	t.Setenv("HOME", t.TempDir())

	_, _, err := ResolveTelemetryToken()
	if err == nil {
		t.Fatal("expected error when no token configured")
	}
	if !strings.Contains(err.Error(), telemetryEnvVarName) {
		t.Errorf("error should mention telemetry env var, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), envVarName) {
		t.Errorf("error should mention uptime env var, got: %s", err.Error())
	}
}

func TestResolveToken_MissingTokenCarriesAuthExitCode(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv("HOME", t.TempDir())

	_, err := ResolveToken()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := errs.CodeOf(err); got != errs.ExitAuth {
		t.Errorf("CodeOf(err) = %d, want %d (ExitAuth)", got, errs.ExitAuth)
	}
}

func TestResolveTelemetryToken_MissingTokenCarriesAuthExitCode(t *testing.T) {
	t.Setenv(envVarName, "")
	t.Setenv(telemetryEnvVarName, "")
	t.Setenv("HOME", t.TempDir())

	_, _, err := ResolveTelemetryToken()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := errs.CodeOf(err); got != errs.ExitAuth {
		t.Errorf("CodeOf(err) = %d, want %d (ExitAuth)", got, errs.ExitAuth)
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
