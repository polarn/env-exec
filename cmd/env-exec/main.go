package main

import (
	"context"
	"fmt"
	"log"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/utils"
)

func main() {
	var envVars = make(map[string]string)

	config := utils.LoadConfig()
	processEnvVars(config, &envVars)
	processEnvVarsGCP(config, &envVars)

	if len(os.Args) < 2 {
		utils.PrintEnvVars(envVars)
	} else {
		utils.SetEnvVars(envVars)
		utils.ExecuteCommand()
	}
}

// Find only normal env vars and add them to the envVars map
func processEnvVars(config *config.RootConfig, envVars *map[string]string) {
	for _, env := range config.Env {
		if env.Value != "" {
			envVarName := utils.MakeProperEnvVarName(env.Name)
			(*envVars)[envVarName] = env.Value
		}
	}
}

// Find GCP secret env vars and add them to the envVars map
func processEnvVarsGCP(config *config.RootConfig, envVars *map[string]string) {
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

			envVarName := utils.MakeProperEnvVarName(env.Name)
			(*envVars)[envVarName] = string(resp.Payload.Data)
		}
	}
}
