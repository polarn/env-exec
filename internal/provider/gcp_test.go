package provider

import (
	"bytes"
	"context"
	"errors"
	"log"
	"maps"
	"os"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
)

type fakeSecretClient struct {
	secrets   map[string]string
	errs      map[string]error
	requested []string
	closed    bool
}

func (f *fakeSecretClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	name := req.GetName()
	f.requested = append(f.requested, name)

	if err, ok := f.errs[name]; ok {
		return nil, err
	}

	value, ok := f.secrets[name]
	if !ok {
		return nil, errors.New("rpc error: code = NotFound")
	}

	return &secretmanagerpb.AccessSecretVersionResponse{
		Name:    name,
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(value)},
	}, nil
}

func (f *fakeSecretClient) Close() error {
	f.closed = true
	return nil
}

func installSecretAccessor(t *testing.T, client secretAccessor, err error) {
	t.Helper()

	original := newSecretAccessor
	newSecretAccessor = func(ctx context.Context) (secretAccessor, error) {
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	t.Cleanup(func() { newSecretAccessor = original })
}

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	flags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
		log.SetFlags(flags)
	})
	return &buf
}

func gcpEnv(name, project, secret, version string) config.EnvConfig {
	return config.EnvConfig{
		Name: name,
		ValueFrom: config.ValueFrom{
			GCPSecretKeyRef: config.GCPSecretKeyRef{
				Project: project,
				Name:    secret,
				Version: version,
			},
		},
	}
}

func TestGCPProvider_Name(t *testing.T) {
	if name := (&GCPProvider{}).Name(); name != "gcp" {
		t.Errorf("want 'gcp', got %q", name)
	}
}

func TestGCPProvider_Provide(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.RootConfig
		secrets       map[string]string
		errs          map[string]error
		existing      map[string]string
		want          map[string]string
		wantRequested []string
		wantLog       string
	}{
		{
			name:          "explicit project and version",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "proj", "api-token", "3")}},
			secrets:       map[string]string{"projects/proj/secrets/api-token/versions/3": "s3cret"},
			want:          map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{"projects/proj/secrets/api-token/versions/3"},
		},
		{
			name:          "empty version defaults to latest",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "proj", "api-token", "")}},
			secrets:       map[string]string{"projects/proj/secrets/api-token/versions/latest": "newest"},
			want:          map[string]string{"TOKEN": "newest"},
			wantRequested: []string{"projects/proj/secrets/api-token/versions/latest"},
		},
		{
			name: "empty project falls back to defaults",
			config: &config.RootConfig{
				Defaults: config.DefaultsConfig{GCP: config.GCPDefaults{Project: "default-proj"}},
				Env:      []config.EnvConfig{gcpEnv("TOKEN", "", "api-token", "1")},
			},
			secrets:       map[string]string{"projects/default-proj/secrets/api-token/versions/1": "from-default"},
			want:          map[string]string{"TOKEN": "from-default"},
			wantRequested: []string{"projects/default-proj/secrets/api-token/versions/1"},
		},
		{
			name: "explicit project wins over defaults",
			config: &config.RootConfig{
				Defaults: config.DefaultsConfig{GCP: config.GCPDefaults{Project: "default-proj"}},
				Env:      []config.EnvConfig{gcpEnv("TOKEN", "own-proj", "api-token", "1")},
			},
			secrets:       map[string]string{"projects/own-proj/secrets/api-token/versions/1": "from-own"},
			want:          map[string]string{"TOKEN": "from-own"},
			wantRequested: []string{"projects/own-proj/secrets/api-token/versions/1"},
		},
		{
			name:          "no project anywhere is skipped",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "", "api-token", "")}},
			want:          map[string]string{},
			wantRequested: nil,
			wantLog:       "Warning: No GCP project found for secret 'TOKEN', skipping",
		},
		{
			name: "missing project skips only that secret",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gcpEnv("NO_PROJECT", "", "orphan", ""),
				gcpEnv("TOKEN", "proj", "api-token", ""),
			}},
			secrets:       map[string]string{"projects/proj/secrets/api-token/versions/latest": "s3cret"},
			want:          map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{"projects/proj/secrets/api-token/versions/latest"},
			wantLog:       "Warning: No GCP project found for secret 'NO_PROJECT', skipping",
		},
		{
			name: "access failure skips only that secret",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gcpEnv("BROKEN", "proj", "broken", ""),
				gcpEnv("TOKEN", "proj", "api-token", ""),
			}},
			secrets: map[string]string{"projects/proj/secrets/api-token/versions/latest": "s3cret"},
			errs: map[string]error{
				"projects/proj/secrets/broken/versions/latest": errors.New("permission denied"),
			},
			want: map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{
				"projects/proj/secrets/broken/versions/latest",
				"projects/proj/secrets/api-token/versions/latest",
			},
			wantLog: "Warning: Failed to access GCP secret 'broken' version 'latest': permission denied",
		},
		{
			name: "ignores plain and gitlab entries",
			config: &config.RootConfig{Env: []config.EnvConfig{
				{Name: "PLAIN", Value: "plain"},
				{Name: "GITLAB", ValueFrom: config.ValueFrom{GitlabVariableKeyRef: config.GitlabVariableKeyRef{Project: "1", Key: "KEY"}}},
				gcpEnv("TOKEN", "proj", "api-token", ""),
			}},
			secrets:       map[string]string{"projects/proj/secrets/api-token/versions/latest": "s3cret"},
			want:          map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{"projects/proj/secrets/api-token/versions/latest"},
		},
		{
			name:          "preserves unrelated keys and overwrites its own",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "proj", "api-token", "")}},
			secrets:       map[string]string{"projects/proj/secrets/api-token/versions/latest": "fresh"},
			existing:      map[string]string{"OTHER": "keep", "TOKEN": "stale"},
			want:          map[string]string{"OTHER": "keep", "TOKEN": "fresh"},
			wantRequested: []string{"projects/proj/secrets/api-token/versions/latest"},
		},
		{
			name:          "payload with newlines is preserved verbatim",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("KEY", "proj", "private-key", "")}},
			secrets:       map[string]string{"projects/proj/secrets/private-key/versions/latest": "-----BEGIN-----\nline\n-----END-----\n"},
			want:          map[string]string{"KEY": "-----BEGIN-----\nline\n-----END-----\n"},
			wantRequested: []string{"projects/proj/secrets/private-key/versions/latest"},
		},
		{
			name:          "empty payload yields empty value",
			config:        &config.RootConfig{Env: []config.EnvConfig{gcpEnv("EMPTY", "proj", "blank", "")}},
			secrets:       map[string]string{"projects/proj/secrets/blank/versions/latest": ""},
			want:          map[string]string{"EMPTY": ""},
			wantRequested: []string{"projects/proj/secrets/blank/versions/latest"},
		},
		{
			name: "multiple secrets fetched in config order",
			config: &config.RootConfig{
				Defaults: config.DefaultsConfig{GCP: config.GCPDefaults{Project: "proj"}},
				Env: []config.EnvConfig{
					gcpEnv("FIRST", "", "one", ""),
					gcpEnv("SECOND", "", "two", "2"),
				},
			},
			secrets: map[string]string{
				"projects/proj/secrets/one/versions/latest": "1",
				"projects/proj/secrets/two/versions/2":      "2",
			},
			want: map[string]string{"FIRST": "1", "SECOND": "2"},
			wantRequested: []string{
				"projects/proj/secrets/one/versions/latest",
				"projects/proj/secrets/two/versions/2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeSecretClient{secrets: tt.secrets, errs: tt.errs}
			installSecretAccessor(t, client, nil)
			logged := captureLog(t)

			envVars := make(map[string]string)
			maps.Copy(envVars, tt.existing)

			p := &GCPProvider{}
			if err := p.Provide(tt.config, envVars); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(envVars, tt.want) {
				t.Errorf("want %v, got %v", tt.want, envVars)
			}
			if !reflect.DeepEqual(client.requested, tt.wantRequested) {
				t.Errorf("want requests %v, got %v", tt.wantRequested, client.requested)
			}
			if !client.closed {
				t.Error("client was not closed")
			}
			if tt.wantLog != "" && !strings.Contains(logged.String(), tt.wantLog) {
				t.Errorf("want log containing %q, got %q", tt.wantLog, logged.String())
			}
			if tt.wantLog == "" && logged.Len() != 0 {
				t.Errorf("want no log output, got %q", logged.String())
			}
		})
	}
}

func TestGCPProvider_ProvideWithoutGCPSecrets(t *testing.T) {
	created := false
	original := newSecretAccessor
	newSecretAccessor = func(ctx context.Context) (secretAccessor, error) {
		created = true
		return nil, errors.New("client should not be created")
	}
	t.Cleanup(func() { newSecretAccessor = original })

	cfg := &config.RootConfig{Env: []config.EnvConfig{
		{Name: "PLAIN", Value: "plain"},
		{Name: "GITLAB", ValueFrom: config.ValueFrom{GitlabVariableKeyRef: config.GitlabVariableKeyRef{Project: "1", Key: "KEY"}}},
	}}

	envVars := map[string]string{"PLAIN": "plain"}
	if err := (&GCPProvider{}).Provide(cfg, envVars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created {
		t.Error("client created for config without GCP secrets")
	}
	if !reflect.DeepEqual(envVars, map[string]string{"PLAIN": "plain"}) {
		t.Errorf("envVars modified: %v", envVars)
	}
}

func TestGCPProvider_ProvideClientError(t *testing.T) {
	wantErr := errors.New("could not find default credentials")
	installSecretAccessor(t, nil, wantErr)

	cfg := &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "proj", "api-token", "")}}

	err := (&GCPProvider{}).Provide(cfg, map[string]string{})
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("want wrapped %v, got %v", wantErr, err)
	}
	if !strings.Contains(err.Error(), "failed to create Secret Manager client") {
		t.Errorf("want client creation message, got %q", err.Error())
	}
}
