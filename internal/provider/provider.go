package provider

import "github.com/polarn/env-exec/internal/config"

// Provider defines the interface for environment variable providers.
type Provider interface {
	Provide(config *config.RootConfig, envVars map[string]string)
}

// PlainProvider provides static environment variables.
type PlainProvider struct{}

// GCPProvider provides environment variables from GCP Secret Manager.
type GCPProvider struct{}

// GitlabProvider provides environment variables from GitLab CI/CD variables.
type GitlabProvider struct{}

// AllProviders returns all available providers in execution order.
func AllProviders() []Provider {
	return []Provider{
		&PlainProvider{},
		&GCPProvider{},
		&GitlabProvider{},
	}
}
