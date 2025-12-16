package env

import (
	"fmt"
	"os"
	"strings"
)

func Print(envVars map[string]string) {
	for key, value := range envVars {
		// Escape single quotes in the value by replacing ' with '\''
		escapedValue := strings.ReplaceAll(value, "'", "'\\''")
		fmt.Printf("export %s='%s'\n", key, escapedValue)
	}
}

func Set(envVars map[string]string) error {
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable '%s': %w", key, err)
		}
	}
	return nil
}
