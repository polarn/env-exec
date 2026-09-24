package config

import (
	"reflect"
	"testing"
)

func TestFileVars(t *testing.T) {
	tests := []struct {
		name   string
		config *RootConfig
		want   []string
	}{
		{"empty config", &RootConfig{}, nil},
		{"no asFile entries", &RootConfig{Env: []EnvConfig{{Name: "A", Value: "1"}}}, nil},
		{
			name: "asFile entries in config order",
			config: &RootConfig{Env: []EnvConfig{
				{Name: "KEY", Value: "k", AsFile: true},
				{Name: "PLAIN", Value: "p"},
				{Name: "CREDS", ValueFrom: ValueFrom{GCPSecretKeyRef: GCPSecretKeyRef{Name: "creds"}}, AsFile: true},
			}},
			want: []string{"KEY", "CREDS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.FileVars(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("want %v, got %v", tt.want, got)
			}
		})
	}
}
