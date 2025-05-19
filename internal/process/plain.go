package process

import "github.com/polarn/env-exec/internal/config"

// Find only normal env vars and add them to the envVars map
func EnvVars(config *config.RootConfig, envVars *map[string]string) {
	for _, env := range config.Env {
		if env.Value != "" {
			(*envVars)[env.Name] = env.Value
		}
	}
}
