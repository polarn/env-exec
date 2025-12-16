package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/polarn/env-exec/internal/config"
)

type githubVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Provide fetches GitHub Actions variables and adds them to the envVars map.
func (p *GithubProvider) Provide(cfg *config.RootConfig, envVars map[string]string) error {
	if !hasGithubVariables(cfg) {
		return nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	for _, env := range cfg.Env {
		if env.ValueFrom.GithubVariableKeyRef.Name != "" {
			name := env.ValueFrom.GithubVariableKeyRef.Name
			repo := env.ValueFrom.GithubVariableKeyRef.Repo

			if repo == "" {
				repo = cfg.Defaults.Github.Repo
			}

			if repo == "" {
				log.Printf("Warning: No GitHub repo found for variable '%s', skipping", env.Name)
				continue
			}

			value, err := getGithubVariable(token, repo, name)
			if err != nil {
				log.Printf("Warning: Failed to get GitHub variable '%s': %v", name, err)
				continue
			}

			envVars[env.Name] = value
		}
	}
	return nil
}

func hasGithubVariables(cfg *config.RootConfig) bool {
	for _, env := range cfg.Env {
		if env.ValueFrom.GithubVariableKeyRef.Name != "" {
			return true
		}
	}
	return false
}

func getGithubVariable(token, repo, name string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/variables/%s", repo, name)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get variable (status %s): %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var variable githubVariable
	if err := json.Unmarshal(bodyBytes, &variable); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	return variable.Value, nil
}
