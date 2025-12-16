package env

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		contains []string // for map iteration order
		exact    string   // for single var tests
	}{
		{
			name:    "empty",
			envVars: map[string]string{},
			exact:   "",
		},
		{
			name:    "single var",
			envVars: map[string]string{"TEST": "hello"},
			exact:   "export TEST='hello'\n",
		},
		{
			name:    "escapes single quotes",
			envVars: map[string]string{"TEST": "it's a test"},
			exact:   "export TEST='it'\\''s a test'\n",
		},
		{
			name:    "multiple single quotes",
			envVars: map[string]string{"TEST": "it's Bob's"},
			exact:   "export TEST='it'\\''s Bob'\\''s'\n",
		},
		{
			name:    "spaces preserved",
			envVars: map[string]string{"TEST": "value with spaces"},
			exact:   "export TEST='value with spaces'\n",
		},
		{
			name:     "multiple vars",
			envVars:  map[string]string{"A": "1", "B": "2"},
			contains: []string{"export A='1'\n", "export B='2'\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() { Print(tt.envVars) })

			if tt.exact != "" {
				if output != tt.exact {
					t.Errorf("want %q, got %q", tt.exact, output)
				}
			}
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("want output to contain %q, got %q", want, output)
				}
			}
		})
	}
}

func TestSet(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
	}{
		{"empty", map[string]string{}},
		{"single var", map[string]string{"TEST_SET_1": "value1"}},
		{"multiple vars", map[string]string{"TEST_SET_A": "a", "TEST_SET_B": "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up after test
			for k := range tt.envVars {
				defer os.Unsetenv(k)
			}

			if err := Set(tt.envVars); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, want := range tt.envVars {
				if got := os.Getenv(k); got != want {
					t.Errorf("want %s=%q, got %q", k, want, got)
				}
			}
		})
	}
}
