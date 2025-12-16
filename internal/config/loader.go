package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func loadFile() ([]byte, error) {
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

func Load() (*RootConfig, error) {
	var cfg RootConfig

	data, err := loadFile()
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
