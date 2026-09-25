package provider

import (
	"context"
	"fmt"
	"slices"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
)

type secretAccessor interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

type secretManagerClient struct {
	client *secretmanager.Client
}

func (c *secretManagerClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return c.client.AccessSecretVersion(ctx, req)
}

func (c *secretManagerClient) Close() error {
	return c.client.Close()
}

var newSecretAccessor = func(ctx context.Context) (secretAccessor, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &secretManagerClient{client: client}, nil
}

var gcpTimeout = time.Minute

// Provide fetches GCP secrets and adds them to the envVars map.
func (p *GCPProvider) Provide(cfg *config.RootConfig, envVars map[string]string) error {
	if !slices.ContainsFunc(cfg.Env, func(env config.EnvConfig) bool { return env.ValueFrom.GCPSecretKeyRef.Name != "" }) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gcpTimeout)
	defer cancel()

	client, err := newSecretAccessor(ctx)
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

			if project == "" {
				project = cfg.Defaults.GCP.Project
			}

			reqName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, name, version)

			resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
				Name: reqName,
			})
			if err != nil {
				return fmt.Errorf("'%s': failed to access %s: %w", env.Name, reqName, err)
			}

			if resp.GetPayload() == nil {
				return fmt.Errorf("'%s': %s returned no payload", env.Name, reqName)
			}

			envVars[env.Name] = string(resp.GetPayload().GetData())
		}
	}
	return nil
}
