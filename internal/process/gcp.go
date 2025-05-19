package process

import (
	"context"
	"fmt"
	"log"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/utils"
)

// Find GCP secret env vars and add them to the envVars map
func EnvVarsGCP(config *config.RootConfig, envVars *map[string]string) {
	if !utils.CheckIfGCPSecretKeyRefExists(config) {
		return
	}

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Secret Manager client: %v", err)
	}
	defer client.Close()

	for _, env := range config.Env {
		if env.ValueFrom.GCPSecretKeyRef.Name != "" {
			name := env.ValueFrom.GCPSecretKeyRef.Name
			version := env.ValueFrom.GCPSecretKeyRef.Version
			project := env.ValueFrom.GCPSecretKeyRef.Project

			if version == "" {
				version = "latest"
			}

			if project == "" && config.Defaults.GCP.Project == "" {
				log.Printf("Error: No GCP project found for secret '%s'", env.Name)
				continue
			} else if project == "" {
				project = config.Defaults.GCP.Project
			}

			reqName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, name, version)

			resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
				Name: reqName,
			})
			if err != nil {
				log.Printf("Error accessing GCP secret '%s' version '%s': %v", name, version, err)
				continue
			}

			(*envVars)[env.Name] = string(resp.Payload.Data)
		}
	}
}
