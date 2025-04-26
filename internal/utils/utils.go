package utils

import (
	"log"
	"os"
	"strings"

	"github.com/polarn/env-exec/internal/config"
	"gopkg.in/yaml.v3"
)

func LoadConfig() *config.RootConfig {
	yamlFile := ".env-exec.yaml"
	if os.Getenv("ENV_EXEC_YAML") != "" {
		yamlFile = os.Getenv("ENV_EXEC_YAML")
	}

	data, err := os.ReadFile(yamlFile)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var config config.RootConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling YAML: %v", err)
	}

	return &config
}

func SanitizeEnvVarName(name string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_")
	sanitized := replacer.Replace(name)
	sanitized = strings.ToUpper(sanitized)
	return sanitized
}
