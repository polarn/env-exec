package provider

import (
	"errors"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/polarn/env-exec/internal/config"
)

type gitlabResponse struct {
	status int
	body   string
}

type fakeGitlab struct {
	mu        sync.Mutex
	responses map[string]gitlabResponse
	block     bool
	requested []string
	tokens    []string
}

func (f *fakeGitlab) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requested = append(f.requested, r.URL.EscapedPath())
	f.tokens = append(f.tokens, r.Header.Get("PRIVATE-TOKEN"))
	resp, ok := f.responses[r.URL.EscapedPath()]
	f.mu.Unlock()

	if f.block {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
		}
		return
	}
	if !ok {
		resp = gitlabResponse{http.StatusNotFound, `{"message":"404 Variable Not Found"}`}
	}
	w.WriteHeader(resp.status)
	io.WriteString(w, resp.body)
}

func (f *fakeGitlab) seen() (requested, tokens []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requested, f.tokens
}

func installGitlab(t *testing.T, handler http.Handler) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := gitlabURL
	gitlabURL = server.URL
	t.Cleanup(func() { gitlabURL = original })
}

func gitlabEnv(name, project, key string) config.EnvConfig {
	return config.EnvConfig{
		Name: name,
		ValueFrom: config.ValueFrom{
			GitlabVariableKeyRef: config.GitlabVariableKeyRef{
				Project: project,
				Key:     key,
			},
		},
	}
}

func TestGitlabProvider_Name(t *testing.T) {
	if name := (&GitlabProvider{}).Name(); name != "gitlab" {
		t.Errorf("want 'gitlab', got %q", name)
	}
}

func TestGitlabProvider_Provide(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.RootConfig
		responses     map[string]gitlabResponse
		existing      map[string]string
		want          map[string]string
		wantRequested []string
		wantErr       string
	}{
		{
			name:   "numeric project ID",
			config: &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "12345", "API_TOKEN")}},
			responses: map[string]gitlabResponse{
				"/api/v4/projects/12345/variables/API_TOKEN": {http.StatusOK, `{"key":"API_TOKEN","value":"s3cret","masked":true}`},
			},
			want:          map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{"/api/v4/projects/12345/variables/API_TOKEN"},
		},
		{
			name:   "project path is escaped",
			config: &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "group/sub/app", "API_TOKEN")}},
			responses: map[string]gitlabResponse{
				"/api/v4/projects/group%2Fsub%2Fapp/variables/API_TOKEN": {http.StatusOK, `{"value":"from-path"}`},
			},
			want:          map[string]string{"TOKEN": "from-path"},
			wantRequested: []string{"/api/v4/projects/group%2Fsub%2Fapp/variables/API_TOKEN"},
		},
		{
			name:   "value with newlines is preserved verbatim",
			config: &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("KEY", "1", "PRIVATE_KEY")}},
			responses: map[string]gitlabResponse{
				"/api/v4/projects/1/variables/PRIVATE_KEY": {http.StatusOK, `{"value":"-----BEGIN-----\nline\n-----END-----\n"}`},
			},
			want:          map[string]string{"KEY": "-----BEGIN-----\nline\n-----END-----\n"},
			wantRequested: []string{"/api/v4/projects/1/variables/PRIVATE_KEY"},
		},
		{
			name: "error status stops at that variable",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gitlabEnv("BROKEN", "1", "FORBIDDEN"),
				gitlabEnv("TOKEN", "1", "API_TOKEN"),
			}},
			responses: map[string]gitlabResponse{
				"/api/v4/projects/1/variables/FORBIDDEN": {http.StatusForbidden, `{"message":"403 Forbidden"}`},
				"/api/v4/projects/1/variables/API_TOKEN": {http.StatusOK, `{"value":"s3cret"}`},
			},
			want:          map[string]string{},
			wantRequested: []string{"/api/v4/projects/1/variables/FORBIDDEN"},
			wantErr:       `'BROKEN': variable 'FORBIDDEN' in project '1': failed to get variable: 403 Forbidden: {"message":"403 Forbidden"}`,
		},
		{
			name:          "invalid JSON",
			config:        &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "1", "API_TOKEN")}},
			responses:     map[string]gitlabResponse{"/api/v4/projects/1/variables/API_TOKEN": {http.StatusOK, `not json`}},
			want:          map[string]string{},
			wantRequested: []string{"/api/v4/projects/1/variables/API_TOKEN"},
			wantErr:       "'TOKEN': variable 'API_TOKEN' in project '1': failed to decode response: invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name: "ignores plain and gcp entries",
			config: &config.RootConfig{Env: []config.EnvConfig{
				{Name: "PLAIN", Value: "plain"},
				gcpEnv("GCP", "proj", "secret", ""),
				gitlabEnv("TOKEN", "1", "API_TOKEN"),
			}},
			responses:     map[string]gitlabResponse{"/api/v4/projects/1/variables/API_TOKEN": {http.StatusOK, `{"value":"s3cret"}`}},
			want:          map[string]string{"TOKEN": "s3cret"},
			wantRequested: []string{"/api/v4/projects/1/variables/API_TOKEN"},
		},
		{
			name:          "preserves unrelated keys and overwrites its own",
			config:        &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "1", "API_TOKEN")}},
			responses:     map[string]gitlabResponse{"/api/v4/projects/1/variables/API_TOKEN": {http.StatusOK, `{"value":"fresh"}`}},
			existing:      map[string]string{"OTHER": "keep", "TOKEN": "stale"},
			want:          map[string]string{"OTHER": "keep", "TOKEN": "fresh"},
			wantRequested: []string{"/api/v4/projects/1/variables/API_TOKEN"},
		},
		{
			name: "multiple variables fetched in config order",
			config: &config.RootConfig{Env: []config.EnvConfig{
				gitlabEnv("FIRST", "1", "ONE"),
				gitlabEnv("SECOND", "group/app", "TWO"),
			}},
			responses: map[string]gitlabResponse{
				"/api/v4/projects/1/variables/ONE":           {http.StatusOK, `{"value":"1"}`},
				"/api/v4/projects/group%2Fapp/variables/TWO": {http.StatusOK, `{"value":"2"}`},
			},
			want: map[string]string{"FIRST": "1", "SECOND": "2"},
			wantRequested: []string{
				"/api/v4/projects/1/variables/ONE",
				"/api/v4/projects/group%2Fapp/variables/TWO",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITLAB_TOKEN", "glpat-test")
			gitlab := &fakeGitlab{responses: tt.responses}
			installGitlab(t, gitlab)

			envVars := make(map[string]string)
			maps.Copy(envVars, tt.existing)

			err := (&GitlabProvider{}).Provide(tt.config, envVars)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr) {
				t.Fatalf("want error %q, got: %v", tt.wantErr, err)
			}

			if !reflect.DeepEqual(envVars, tt.want) {
				t.Errorf("want %v, got %v", tt.want, envVars)
			}
			requested, tokens := gitlab.seen()
			if !reflect.DeepEqual(requested, tt.wantRequested) {
				t.Errorf("want requests %v, got %v", tt.wantRequested, requested)
			}
			for _, token := range tokens {
				if token != "glpat-test" {
					t.Errorf("want PRIVATE-TOKEN 'glpat-test', got %q", token)
				}
			}
		})
	}
}

func TestGitlabProvider_ProvideWithoutGitlabVariables(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	gitlab := &fakeGitlab{}
	installGitlab(t, gitlab)

	cfg := &config.RootConfig{Env: []config.EnvConfig{
		{Name: "PLAIN", Value: "plain"},
		gcpEnv("GCP", "proj", "secret", ""),
	}}

	envVars := map[string]string{"PLAIN": "plain"}
	if err := (&GitlabProvider{}).Provide(cfg, envVars); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requested, _ := gitlab.seen(); len(requested) != 0 {
		t.Errorf("requests made for config without GitLab variables: %v", requested)
	}
	if !reflect.DeepEqual(envVars, map[string]string{"PLAIN": "plain"}) {
		t.Errorf("envVars modified: %v", envVars)
	}
}

func TestGitlabProvider_ProvideWithoutToken(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	gitlab := &fakeGitlab{}
	installGitlab(t, gitlab)

	cfg := &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "1", "API_TOKEN")}}

	err := (&GitlabProvider{}).Provide(cfg, map[string]string{})
	if err == nil || err.Error() != "GITLAB_TOKEN environment variable not set" {
		t.Fatalf("want missing token error, got %v", err)
	}
	if requested, _ := gitlab.seen(); len(requested) != 0 {
		t.Errorf("requests made without a token: %v", requested)
	}
}

func TestGitlabProvider_ProvideTimeout(t *testing.T) {
	original := gitlabTimeout
	gitlabTimeout = 10 * time.Millisecond
	t.Cleanup(func() { gitlabTimeout = original })

	t.Setenv("GITLAB_TOKEN", "glpat-test")
	installGitlab(t, &fakeGitlab{block: true})

	cfg := &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "1", "API_TOKEN")}}

	err := (&GitlabProvider{}).Provide(cfg, map[string]string{})
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("want timeout, got %v", err)
	}
}

func TestGitlabProvider_ProvideTruncatedErrorBody(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "glpat-test")
	installGitlab(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "short")
	}))

	cfg := &config.RootConfig{Env: []config.EnvConfig{gitlabEnv("TOKEN", "1", "API_TOKEN")}}

	err := (&GitlabProvider{}).Provide(cfg, map[string]string{})
	want := "'TOKEN': variable 'API_TOKEN' in project '1': failed to get variable: 502 Bad Gateway (reading body: unexpected EOF)"
	if err == nil || err.Error() != want {
		t.Fatalf("want error %q, got: %v", want, err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("want wrapped %v, got %v", io.ErrUnexpectedEOF, err)
	}
}
