package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/utils"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func main() {
	config := utils.LoadConfig()
	envVars := processEnvVars(config)

	if len(os.Args) < 2 {
		utils.PrintEnvVars(envVars)
	} else {
		utils.SetEnvVars(envVars)
		utils.ExecuteCommand()
	}
}

func processEnvVars(config *config.RootConfig) map[string]string {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create Secret Manager client: %v", err)
	}
	defer client.Close()

	envVars := make(map[string]string)

	for _, env := range config.Env {
		envVarName := utils.MakeProperEnvVarName(env.Name)
		envVarValue := ""

		if env.Value != "" {
			envVarValue = env.Value
		} else if env.ValueFrom.GCPSecretKeyRef.Name != "" {
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

			envVarValue = string(resp.Payload.Data)
		}

		envVars[envVarName] = envVarValue
	}
	return envVars
}
