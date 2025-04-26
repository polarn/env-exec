package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/utils"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func main() {
	config := utils.LoadConfig()
	envVars := processEnvVars(config)

	if len(os.Args) < 2 {
		outputEnvVars(envVars)
	} else {
		setEnvVars(envVars)
		executeCommand()
	}
}

func outputEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		fmt.Printf("export %s=\"%s\"\n", key, value)
	}
}

func setEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		err := os.Setenv(key, value)
		if err != nil {
			fmt.Println("Error setting environment variable:", err)
			return
		}
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
		envVarName := utils.SanitizeEnvVarName(env.Name)
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

func executeCommand() error {
	command := os.Args[1]
	args := os.Args[2:]

	cmd := exec.Command(command, args...)

	// Set the standard input, output, and error streams to the current process's
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// If the command exited with a non-zero status, the error will be of type *exec.ExitError
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Command exited with status: %d\n", status.ExitStatus())
			} else {
				fmt.Printf("Command failed: %v\n", err)
			}
			os.Exit(1)
		} else {
			fmt.Printf("Failed to run command: %v\n", err)
			os.Exit(1)
		}
	}
	return cmd.Run()
}
