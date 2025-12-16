package provider

import (
	"testing"

	"github.com/polarn/env-exec/internal/config"
)

func TestPlainProvider_Name(t *testing.T) {
	if name := (&PlainProvider{}).Name(); name != "plain" {
		t.Errorf("want 'plain', got %q", name)
	}
}

func TestPlainProvider_Provide(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.RootConfig
		existing map[string]string
		want     map[string]string
	}{
		{
			name:   "empty config",
			config: &config.RootConfig{},
			want:   map[string]string{},
		},
		{
			name:   "single value",
			config: &config.RootConfig{Env: []config.EnvConfig{{Name: "TEST", Value: "hello"}}},
			want:   map[string]string{"TEST": "hello"},
		},
		{
			name: "multiple values",
			config: &config.RootConfig{Env: []config.EnvConfig{
				{Name: "A", Value: "1"},
				{Name: "B", Value: "2"},
			}},
			want: map[string]string{"A": "1", "B": "2"},
		},
		{
			name: "skips empty value",
			config: &config.RootConfig{Env: []config.EnvConfig{
				{Name: "EMPTY", Value: ""},
				{Name: "SET", Value: "value"},
			}},
			want: map[string]string{"SET": "value"},
		},
		{
			name: "skips valueFrom entries",
			config: &config.RootConfig{Env: []config.EnvConfig{
				{Name: "PLAIN", Value: "plain"},
				{Name: "GCP", ValueFrom: config.ValueFrom{GCPSecretKeyRef: config.GCPSecretKeyRef{Name: "secret"}}},
			}},
			want: map[string]string{"PLAIN": "plain"},
		},
		{
			name:     "preserves existing",
			config:   &config.RootConfig{Env: []config.EnvConfig{{Name: "NEW", Value: "new"}}},
			existing: map[string]string{"OLD": "old"},
			want:     map[string]string{"OLD": "old", "NEW": "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envVars := make(map[string]string)
			for k, v := range tt.existing {
				envVars[k] = v
			}

			p := &PlainProvider{}
			if err := p.Provide(tt.config, envVars); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(envVars) != len(tt.want) {
				t.Errorf("want %d vars, got %d: %v", len(tt.want), len(envVars), envVars)
			}
			for k, v := range tt.want {
				if envVars[k] != v {
					t.Errorf("want %s=%q, got %q", k, v, envVars[k])
				}
			}
		})
	}
}
