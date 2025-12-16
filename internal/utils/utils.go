package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/polarn/env-exec/internal/config"
	"gopkg.in/yaml.v3"
)

func LoadConfigFile() ([]byte, error) {
	filename := ".env-exec.yaml"
	if os.Getenv("ENV_EXEC_YAML") != "" {
		filename = os.Getenv("ENV_EXEC_YAML")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// This is OK, means we will not expose any variables
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", filename, err)
	}
	return data, nil
}

func LoadConfig() (*config.RootConfig, error) {
	var cfg config.RootConfig

	data, err := LoadConfigFile()
	if err != nil {
		return nil, err
	}
	if data != nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
	}
	return &cfg, nil
}

func PrintEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		// Escape single quotes in the value by replacing ' with '\''
		escapedValue := strings.ReplaceAll(value, "'", "'\\''")
		fmt.Printf("export %s='%s'\n", key, escapedValue)
	}
}

func SetEnvVars(envVars map[string]string) error {
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable '%s': %w", key, err)
		}
	}
	return nil
}

func ExecuteCommand() error {
	command := os.Args[1]
	args := os.Args[2:]

	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("failed to run command: %w", err)
	}
	return nil
}
