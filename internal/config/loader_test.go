package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantErr    bool
		wantEnvLen int
		check      func(*testing.T, *RootConfig)
	}{
		{
			name:       "valid plain value",
			content:    "env:\n  - name: TEST\n    value: hello",
			wantEnvLen: 1,
			check: func(t *testing.T, cfg *RootConfig) {
				if cfg.Env[0].Name != "TEST" || cfg.Env[0].Value != "hello" {
					t.Errorf("got %+v", cfg.Env[0])
				}
			},
		},
		{
			name:       "with defaults",
			content:    "defaults:\n  gcp:\n    project: my-project\nenv:\n  - name: X\n    value: y",
			wantEnvLen: 1,
			check: func(t *testing.T, cfg *RootConfig) {
				if cfg.Defaults.GCP.Project != "my-project" {
					t.Errorf("want default project 'my-project', got %q", cfg.Defaults.GCP.Project)
				}
			},
		},
		{
			name: "GCP secret",
			content: `env:
  - name: SECRET
    valueFrom:
      gcpSecretKeyRef:
        name: my-secret
        project: my-project`,
			wantEnvLen: 1,
			check: func(t *testing.T, cfg *RootConfig) {
				if cfg.Env[0].ValueFrom.GCPSecretKeyRef.Name != "my-secret" {
					t.Errorf("got %+v", cfg.Env[0].ValueFrom.GCPSecretKeyRef)
				}
			},
		},
		{
			name: "GitLab variable",
			content: `env:
  - name: VAR
    valueFrom:
      gitlabVariableKeyRef:
        project: "12345"
        key: my-key`,
			wantEnvLen: 1,
			check: func(t *testing.T, cfg *RootConfig) {
				ref := cfg.Env[0].ValueFrom.GitlabVariableKeyRef
				if ref.Project != "12345" || ref.Key != "my-key" {
					t.Errorf("got %+v", ref)
				}
			},
		},
		{
			name:       "empty file",
			content:    "",
			wantEnvLen: 0,
		},
		{
			name:    "invalid YAML",
			content: "env:\n  - name: TEST\n    value: [invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ENV_EXEC_YAML", configPath)

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Error("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cfg.Env) != tt.wantEnvLen {
				t.Errorf("want %d env vars, got %d", tt.wantEnvLen, len(cfg.Env))
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoad_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)
	os.Unsetenv("ENV_EXEC_YAML")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("want no error for missing file, got: %v", err)
	}
	if cfg == nil || len(cfg.Env) != 0 {
		t.Errorf("want empty config, got: %+v", cfg)
	}
}
