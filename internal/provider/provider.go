package provider

import "github.com/polarn/env-exec/internal/config"

// Provider defines the interface for environment variable providers.
type Provider interface {
	Name() string
	Provide(config *config.RootConfig, envVars map[string]string) error
}

// PlainProvider provides static environment variables.
type PlainProvider struct{}

func (p *PlainProvider) Name() string { return "plain" }

// GCPProvider provides environment variables from GCP Secret Manager.
type GCPProvider struct{}

func (p *GCPProvider) Name() string { return "gcp" }

// GitlabProvider provides environment variables from GitLab CI/CD variables.
type GitlabProvider struct{}

func (p *GitlabProvider) Name() string { return "gitlab" }

// GithubProvider provides environment variables from GitHub Actions variables.
type GithubProvider struct{}

func (p *GithubProvider) Name() string { return "github" }

// AllProviders returns all available providers in execution order.
func AllProviders() []Provider {
	return []Provider{
		&PlainProvider{},
		&GCPProvider{},
		&GitlabProvider{},
		&GithubProvider{},
	}
}
