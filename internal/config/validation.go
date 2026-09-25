package config

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Validate(cfg *RootConfig) error {
	seen := make(map[string]int)

	for i, env := range cfg.Env {
		prefix := fmt.Sprintf("env[%d]", i)
		if env.Name != "" {
			prefix = fmt.Sprintf("env[%d] '%s'", i, env.Name)
		}

		if env.Name == "" {
			return fmt.Errorf("%s: name is required", prefix)
		}

		if !validName.MatchString(env.Name) {
			return fmt.Errorf("%s: name must match %s", prefix, validName)
		}

		if first, ok := seen[env.Name]; ok {
			return fmt.Errorf("%s: duplicate name, first defined at env[%d]", prefix, first)
		}
		seen[env.Name] = i

		hasValue := env.Value != ""
		hasGCP := env.ValueFrom.GCPSecretKeyRef.Name != ""
		hasGitlab := env.ValueFrom.GitlabVariableKeyRef.Key != ""
		hasValueFrom := hasGCP || hasGitlab

		if !hasValue && !hasValueFrom {
			return fmt.Errorf("%s: must have value or valueFrom", prefix)
		}

		if hasValue && hasValueFrom {
			return fmt.Errorf("%s: value and valueFrom are mutually exclusive", prefix)
		}

		if hasGCP && hasGitlab {
			return fmt.Errorf("%s: gcpSecretKeyRef and gitlabVariableKeyRef are mutually exclusive", prefix)
		}

		if hasGCP && env.ValueFrom.GCPSecretKeyRef.Project == "" && cfg.Defaults.GCP.Project == "" {
			return fmt.Errorf("%s: gcpSecretKeyRef.project is required when defaults.gcp.project is not set", prefix)
		}

		if hasGitlab && env.ValueFrom.GitlabVariableKeyRef.Project == "" {
			return fmt.Errorf("%s: gitlabVariableKeyRef.project is required", prefix)
		}
	}

	return nil
}
