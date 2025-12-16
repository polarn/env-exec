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
				ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Name: "secret"}},
			}}},
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
