package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/polarn/env-exec/internal/config"
)

var (
	gitlabURL     = "https://gitlab.com"
	gitlabTimeout = 30 * time.Second
)

// Provide fetches GitLab variables and adds them to the envVars map.
func (p *GitlabProvider) Provide(cfg *config.RootConfig, envVars map[string]string) error {
	if !slices.ContainsFunc(cfg.Env, func(env config.EnvConfig) bool { return env.ValueFrom.GitlabVariableKeyRef.Key != "" }) {
		return nil
	}

	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		return fmt.Errorf("GITLAB_TOKEN environment variable not set")
	}

	client := &http.Client{Timeout: gitlabTimeout}
	for _, env := range cfg.Env {
		if env.ValueFrom.GitlabVariableKeyRef.Key != "" {
			key := env.ValueFrom.GitlabVariableKeyRef.Key
			project := env.ValueFrom.GitlabVariableKeyRef.Project

			value, err := getGitlabVariable(client, gitlabToken, key, project)
			if err != nil {
				return fmt.Errorf("'%s': variable '%s' in project '%s': %w", env.Name, key, project, err)
			}

			envVars[env.Name] = value
		}
	}
	return nil
}

func getGitlabVariable(client *http.Client, gitlabToken, key, project string) (string, error) {
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/variables/%s", gitlabURL, url.PathEscape(project), url.PathEscape(key))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", gitlabToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to get variable: %s (reading body: %w)", resp.Status, err)
		}
		return "", fmt.Errorf("failed to get variable: %s: %s", resp.Status, body)
	}

	var variable struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&variable); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return variable.Value, nil
}
