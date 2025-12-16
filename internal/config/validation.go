package config

import (
	"fmt"
	"log"
)

func Validate(cfg *RootConfig) error {
	seen := make(map[string]bool)

	for i, env := range cfg.Env {
		prefix := fmt.Sprintf("env[%d]", i)
		if env.Name != "" {
			prefix = fmt.Sprintf("env[%d] '%s'", i, env.Name)
		}

		// Name is required
		if env.Name == "" {
			return fmt.Errorf("%s: name is required", prefix)
		}

		// Check for duplicates
		if seen[env.Name] {
			log.Printf("Warning: %s: duplicate env name", prefix)
		}
		seen[env.Name] = true

		// Must have value or valueFrom, not neither
		hasValue := env.Value != ""
		hasGCP := env.ValueFrom.GCPSecretKeyRef.Name != ""
		hasGitlab := env.ValueFrom.GitlabVariableKeyRef.Key != ""
		hasGithub := env.ValueFrom.GithubVariableKeyRef.Name != ""
		hasValueFrom := hasGCP || hasGitlab || hasGithub

		if !hasValue && !hasValueFrom {
			return fmt.Errorf("%s: must have value or valueFrom", prefix)
		}

		// Warn if both value and valueFrom are set
		if hasValue && hasValueFrom {
			log.Printf("Warning: %s: has both value and valueFrom, value takes precedence", prefix)
		}

		// Validate GCP secret ref
		if hasGCP {
			if env.ValueFrom.GCPSecretKeyRef.Name == "" {
				return fmt.Errorf("%s: gcpSecretKeyRef.name is required", prefix)
			}
		}

		// Validate GitLab variable ref
		if hasGitlab {
			if env.ValueFrom.GitlabVariableKeyRef.Key == "" {
				return fmt.Errorf("%s: gitlabVariableKeyRef.key is required", prefix)
			}
			if env.ValueFrom.GitlabVariableKeyRef.Project == "" {
				return fmt.Errorf("%s: gitlabVariableKeyRef.project is required", prefix)
			}
		}

		// Validate GitHub variable ref
		if hasGithub {
			if env.ValueFrom.GithubVariableKeyRef.Name == "" {
				return fmt.Errorf("%s: githubVariableKeyRef.name is required", prefix)
			}
			// repo is optional if defaults.github.repo is set (checked at runtime)
		}
	}

	return nil
}
