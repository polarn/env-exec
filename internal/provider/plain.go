package provider

import "github.com/polarn/env-exec/internal/config"

// Provide adds plain env vars to the envVars map.
func (p *PlainProvider) Provide(cfg *config.RootConfig, envVars map[string]string) error {
	for _, env := range cfg.Env {
		if env.Value != "" {
			envVars[env.Name] = env.Value
		}
	}
	return nil
}
