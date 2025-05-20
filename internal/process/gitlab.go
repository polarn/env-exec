package process

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/polarn/env-exec/internal/config"
)

type GitlabVariable struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	VariableType     string `json:"variable_type"`
	Protected        bool   `json:"protected"`
	Masked           bool   `json:"masked"`
	EnvironmentScope string `json:"environment_scope"`
}

func EnvVarsGitlab(config *config.RootConfig, envVars *map[string]string) {
	if !checkIfGitlabVariableKeyRefExists(config) {
		return
	}

	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		log.Fatal("GITLAB_TOKEN environment variable not set")
	}

	for _, env := range config.Env {
		if env.ValueFrom.GitlabVariableKeyRef.Key != "" {
			key := env.ValueFrom.GitlabVariableKeyRef.Key
			project := env.ValueFrom.GitlabVariableKeyRef.Project

			if project == "" {
				log.Printf("Error: No GitLab project found for variable '%s'", env.Name)
				continue
			}

			value, err := getGitlabVariable(gitlabToken, key, project)
			if err != nil {
				log.Printf("Error getting GitLab variable '%s': %v", key, err)
				continue
			}

			(*envVars)[env.Name] = value
		}
	}
}

func checkIfGitlabVariableKeyRefExists(config *config.RootConfig) bool {
	for _, env := range config.Env {
		if env.ValueFrom.GitlabVariableKeyRef.Key != "" {
			return true
		}
	}
	return false
}

func getGitlabVariable(gitlabToken, key, project string) (string, error) {

	apiURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/variables/%s", project, key)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("PRIVATE-TOKEN", gitlabToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Failed to get variable. Status: %s, Body: %s", resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %w", err)
	}

	var variable GitlabVariable
	if err := json.Unmarshal(bodyBytes, &variable); err != nil {
		return "", fmt.Errorf("Failed to unmarshal JSON response: %w", err)
	}

	return variable.Value, nil
}
