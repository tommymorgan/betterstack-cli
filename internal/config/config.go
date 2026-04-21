package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/tommymorgan/betterstack-cli/internal/errs"
)

const (
	envVarName          = "BETTERSTACK_API_TOKEN"
	telemetryEnvVarName = "BETTERSTACK_TELEMETRY_TOKEN"
	configDir           = "betterstack"
	configFileName      = "config.yaml"
	filePerms           = fs.FileMode(0o600)
)

type configFile struct {
	APIToken          string `yaml:"api_token"`
	TelemetryAPIToken string `yaml:"telemetry_api_token"`
}

// TokenSource describes how a resolved token was located. Callers use this to
// decide whether to emit the telemetry-fallback TTY notice.
type TokenSource int

const (
	// SourceTelemetryEnv: resolved from BETTERSTACK_TELEMETRY_TOKEN env var.
	SourceTelemetryEnv TokenSource = iota
	// SourceTelemetryFile: resolved from telemetry_api_token in the config file.
	SourceTelemetryFile
	// SourceUptimeFallback: no telemetry token found; falling back to the
	// uptime token (BETTERSTACK_API_TOKEN or api_token in the config file).
	SourceUptimeFallback
)

func ResolveToken() (string, error) {
	if token := os.Getenv(envVarName); token != "" {
		return token, nil
	}

	cfg, path, err := loadConfig()
	if err != nil {
		return "", err
	}
	if cfg != nil && cfg.APIToken != "" {
		return cfg.APIToken, nil
	}
	_ = path // unused when the happy path through env var succeeded
	return "", noTokenError()
}

// ResolveTelemetryToken prefers the telemetry token, falling back to the
// uptime token if the telemetry token is not configured. The returned source
// tells the caller how the value was located so they can emit the TTY notice.
func ResolveTelemetryToken() (string, TokenSource, error) {
	if token := os.Getenv(telemetryEnvVarName); token != "" {
		return token, SourceTelemetryEnv, nil
	}

	cfg, _, err := loadConfig()
	if err != nil {
		return "", 0, err
	}
	if cfg != nil && cfg.TelemetryAPIToken != "" {
		return cfg.TelemetryAPIToken, SourceTelemetryFile, nil
	}

	// Fall back to uptime token resolution
	if token := os.Getenv(envVarName); token != "" {
		return token, SourceUptimeFallback, nil
	}
	if cfg != nil && cfg.APIToken != "" {
		return cfg.APIToken, SourceUptimeFallback, nil
	}

	return "", 0, noTelemetryTokenError()
}

// loadConfig reads and parses the config file once, applying the 0600
// permissions warning. Returns (nil, path, nil) when the file does not exist.
func loadConfig() (*configFile, string, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, "", err
	}

	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, path, nil
	}
	if err != nil {
		return nil, path, fmt.Errorf("cannot access config file %s: %w", path, err)
	}

	if info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: %s has permissions %o, expected 0600. Fix with: chmod 600 %s\n", path, info.Mode().Perm(), path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, path, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &cfg, path, nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDir, configFileName), nil
}

// noTokenError reports a missing uptime API token as an auth-failure CLI
// error (exit code 2) so the main entrypoint can map it to the correct exit
// code without every callsite wrapping it.
func noTokenError() error {
	return errs.New(errs.ExitAuth,
		"no API token found. Set %s or create ~/.config/%s/%s with:\n  api_token: <your-token>",
		envVarName, configDir, configFileName)
}

func noTelemetryTokenError() error {
	return errs.New(errs.ExitAuth,
		"no telemetry token found. Set %s or %s, or add telemetry_api_token or api_token to ~/.config/%s/%s. "+
			"Telemetry commands prefer %s then telemetry_api_token, and fall back to %s/api_token (works only for global tokens).",
		telemetryEnvVarName, envVarName, configDir, configFileName,
		telemetryEnvVarName, envVarName,
	)
}
