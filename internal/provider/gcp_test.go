package provider

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/polarn/env-exec/internal/config"
)

type fakeSecretClient struct {
	secrets   map[string]string
	errs      map[string]error
	noPayload map[string]bool
	block     bool
	requested []string
	closed    bool
}

func (f *fakeSecretClient) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	name := req.GetName()
	f.requested = append(f.requested, name)

	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	if err, ok := f.errs[name]; ok {
		return nil, err
	}

	if f.noPayload[name] {
		return &secretmanagerpb.AccessSecretVersionResponse{Name: name}, nil
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
		noPayload     map[string]bool
		existing      map[string]string
		want          map[string]string
		wantRequested []string
		wantErr       string
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
			name: "access failure stops at that secret",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gcpEnv("BROKEN", "proj", "broken", ""),
				gcpEnv("TOKEN", "proj", "api-token", ""),
			}},
			secrets: map[string]string{"projects/proj/secrets/api-token/versions/latest": "s3cret"},
			errs: map[string]error{
				"projects/proj/secrets/broken/versions/latest": errors.New("permission denied"),
			},
			want:          map[string]string{},
			wantRequested: []string{"projects/proj/secrets/broken/versions/latest"},
			wantErr:       "'BROKEN': failed to access projects/proj/secrets/broken/versions/latest: permission denied",
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
			name: "missing payload stops at that secret",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gcpEnv("TOKEN", "proj", "api-token", ""),
				gcpEnv("EMPTY", "proj", "no-payload", "7"),
				gcpEnv("LATER", "proj", "later", ""),
			}},
			secrets: map[string]string{
				"projects/proj/secrets/api-token/versions/latest": "s3cret",
				"projects/proj/secrets/later/versions/latest":     "later",
			},
			noPayload: map[string]bool{"projects/proj/secrets/no-payload/versions/7": true},
			want:      map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{
				"projects/proj/secrets/api-token/versions/latest",
				"projects/proj/secrets/no-payload/versions/7",
			},
			wantErr: "'EMPTY': projects/proj/secrets/no-payload/versions/7 returned no payload",
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
			client := &fakeSecretClient{secrets: tt.secrets, errs: tt.errs, noPayload: tt.noPayload}
			installSecretAccessor(t, client, nil)

			envVars := make(map[string]string)
			maps.Copy(envVars, tt.existing)

			err := (&GCPProvider{}).Provide(tt.config, envVars)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("want error %q, got: %v", tt.wantErr, err)
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

func TestGCPProvider_ProvideTimeout(t *testing.T) {
	original := gcpTimeout
	gcpTimeout = 10 * time.Millisecond
	t.Cleanup(func() { gcpTimeout = original })

	client := &fakeSecretClient{block: true}
	installSecretAccessor(t, client, nil)

	cfg := &config.RootConfig{Env: []config.EnvConfig{gcpEnv("TOKEN", "proj", "api-token", "")}}

	err := (&GCPProvider{}).Provide(cfg, map[string]string{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline exceeded, got %v", err)
	}
	if !client.closed {
		t.Error("client was not closed")
	}
}
