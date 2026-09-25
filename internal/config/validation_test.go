package config

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RootConfig
		wantErr string
	}{
		{
			name:    "empty config",
			config:  &RootConfig{},
			wantErr: "",
		},
		{
			name:    "valid plain value",
			config:  &RootConfig{Env: []EnvConfig{{Name: "TEST", Value: "hello"}}},
			wantErr: "",
		},
		{
			name: "valid GCP secret",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Project: "proj", Name: "secret"}},
			}}},
			wantErr: "",
		},
		{
			name: "GCP project from defaults",
			config: &RootConfig{
				Defaults: DefaultsConfig{GCP: GCPDefaults{Project: "proj"}},
				Env: []EnvConfig{{
					Name:      "TEST",
					ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Name: "secret"}},
				}},
			},
			wantErr: "",
		},
		{
			name: "valid GitLab variable",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				ValueFrom: ValueFrom{GitlabVariableKeyRef: GitlabVariableKeyRef{Project: "123", Key: "key"}},
			}}},
			wantErr: "",
		},
		{
			name: "multiple valid env vars",
			config: &RootConfig{Env: []EnvConfig{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			}},
			wantErr: "",
		},
		{
			name: "names with underscores, lowercase and digits",
			config: &RootConfig{Env: []EnvConfig{
				{Name: "_PRIVATE", Value: "1"},
				{Name: "lower_case", Value: "2"},
				{Name: "VAR_2", Value: "3"},
			}},
			wantErr: "",
		},
		{
			name:    "missing name",
			config:  &RootConfig{Env: []EnvConfig{{Value: "test"}}},
			wantErr: "name is required",
		},
		{
			name:    "missing value and valueFrom",
			config:  &RootConfig{Env: []EnvConfig{{Name: "TEST"}}},
			wantErr: "must have value or valueFrom",
		},
		{
			name:    "name with space",
			config:  &RootConfig{Env: []EnvConfig{{Name: "FOO BAR", Value: "x"}}},
			wantErr: "env[0] 'FOO BAR': name must match",
		},
		{
			name:    "name with dash",
			config:  &RootConfig{Env: []EnvConfig{{Name: "FOO-BAR", Value: "x"}}},
			wantErr: "name must match",
		},
		{
			name:    "name starting with digit",
			config:  &RootConfig{Env: []EnvConfig{{Name: "1FOO", Value: "x"}}},
			wantErr: "name must match",
		},
		{
			name:    "name with equals sign",
			config:  &RootConfig{Env: []EnvConfig{{Name: "FOO=BAR", Value: "x"}}},
			wantErr: "name must match",
		},
		{
			name:    "name with path separator",
			config:  &RootConfig{Env: []EnvConfig{{Name: "../KEY", Value: "x", AsFile: true}}},
			wantErr: "name must match",
		},
		{
			name: "duplicate name",
			config: &RootConfig{Env: []EnvConfig{
				{Name: "A", Value: "1"},
				{Name: "B", Value: "2"},
				{Name: "A", Value: "3"},
			}},
			wantErr: "env[2] 'A': duplicate name, first defined at env[0]",
		},
		{
			name: "value and GCP secret",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				Value:     "fallback",
				ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Project: "proj", Name: "secret"}},
			}}},
			wantErr: "value and valueFrom are mutually exclusive",
		},
		{
			name: "value and GitLab variable",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				Value:     "fallback",
				ValueFrom: ValueFrom{GitlabVariableKeyRef: GitlabVariableKeyRef{Project: "123", Key: "key"}},
			}}},
			wantErr: "value and valueFrom are mutually exclusive",
		},
		{
			name: "GCP secret and GitLab variable",
			config: &RootConfig{Env: []EnvConfig{{
				Name: "TEST",
				ValueFrom: ValueFrom{
					GCPSecretKeyRef:      GCPSecretKeyRef{Project: "proj", Name: "secret"},
					GitlabVariableKeyRef: GitlabVariableKeyRef{Project: "123", Key: "key"},
				},
			}}},
			wantErr: "gcpSecretKeyRef and gitlabVariableKeyRef are mutually exclusive",
		},
		{
			name: "GCP missing project without defaults",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Name: "secret"}},
			}}},
			wantErr: "gcpSecretKeyRef.project is required when defaults.gcp.project is not set",
		},
		{
			name: "GitLab missing project",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				ValueFrom: ValueFrom{GitlabVariableKeyRef: GitlabVariableKeyRef{Key: "key"}},
			}}},
			wantErr: "gitlabVariableKeyRef.project is required",
		},
		{
			name: "GitLab missing key treated as no valueFrom",
			config: &RootConfig{Env: []EnvConfig{{
				Name:      "TEST",
				ValueFrom: ValueFrom{GitlabVariableKeyRef: GitlabVariableKeyRef{Project: "123"}},
			}}},
			wantErr: "must have value or valueFrom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.config)
			if tt.wantErr == "" && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Errorf("want error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
