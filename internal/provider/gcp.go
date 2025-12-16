package provider

import (
	"context"
	"fmt"
	"log"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
)

// Provide fetches GCP secrets and adds them to the envVars map.
func (p *GCPProvider) Provide(cfg *config.RootConfig, envVars map[string]string) error {
	if !hasGCPSecrets(cfg) {
		return nil
	}

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Secret Manager client: %w", err)
	}
	defer client.Close()

	for _, env := range cfg.Env {
		if env.ValueFrom.GCPSecretKeyRef.Name != "" {
			name := env.ValueFrom.GCPSecretKeyRef.Name
			version := env.ValueFrom.GCPSecretKeyRef.Version
			project := env.ValueFrom.GCPSecretKeyRef.Project

			if version == "" {
				version = "latest"
			}

			if project == "" && cfg.Defaults.GCP.Project == "" {
				log.Printf("Warning: No GCP project found for secret '%s', skipping", env.Name)
				continue
			} else if project == "" {
				project = cfg.Defaults.GCP.Project
			}

			reqName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, name, version)

			resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
				Name: reqName,
			})
			if err != nil {
				log.Printf("Warning: Failed to access GCP secret '%s' version '%s': %v", name, version, err)
				continue
			}

			envVars[env.Name] = string(resp.Payload.Data)
		}
	}
	return nil
}

func hasGCPSecrets(cfg *config.RootConfig) bool {
	for _, env := range cfg.Env {
		if env.ValueFrom.GCPSecretKeyRef.Name != "" {
			return true
		}
	}
	return false
}
