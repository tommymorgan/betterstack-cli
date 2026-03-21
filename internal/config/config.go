package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	envVarName     = "BETTERSTACK_API_TOKEN"
	configDir      = "betterstack"
	configFileName = "config.yaml"
	filePerms      = fs.FileMode(0o600)
)

type configFile struct {
	APIToken string `yaml:"api_token"`
}

func ResolveToken() (string, error) {
	if token := os.Getenv(envVarName); token != "" {
		return token, nil
	}

	return readTokenFromFile()
}

func readTokenFromFile() (string, error) {
	path, err := configFilePath()
	if err != nil {
		return "", noTokenError()
	}

	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", noTokenError()
	}
	if err != nil {
		return "", fmt.Errorf("cannot access config file %s: %w", path, err)
	}

	if info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "Warning: %s has permissions %o, expected 0600. Fix with: chmod 600 %s\n", path, info.Mode().Perm(), path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", noTokenError()
	}

	var cfg configFile
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.APIToken == "" {
		return "", noTokenError()
	}

	return cfg.APIToken, nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", configDir, configFileName), nil
}

func noTokenError() error {
	return fmt.Errorf("no API token found. Set %s or create ~/.config/%s/%s with:\n  api_token: <your-token>", envVarName, configDir, configFileName)
}
