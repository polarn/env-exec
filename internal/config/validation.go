package config

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Validate(cfg *RootConfig) error {
	seen := make(map[string]bool)

	for i, env := range cfg.Env {
		prefix := fmt.Sprintf("env[%d]", i)
		if env.Name != "" {
			prefix = fmt.Sprintf("env[%d] '%s'", i, env.Name)
		}

		if env.Name == "" {
			return fmt.Errorf("%s: name is required", prefix)
		}

		if !validName.MatchString(env.Name) {
			return fmt.Errorf("%s: invalid name", prefix)
		}

		if seen[env.Name] {
			return fmt.Errorf("%s: duplicate env name", prefix)
		}
		seen[env.Name] = true

		hasValue := env.Value != ""
		hasGCP := env.ValueFrom.GCPSecretKeyRef.Name != ""
		hasGitlab := env.ValueFrom.GitlabVariableKeyRef.Key != ""
		hasValueFrom := hasGCP || hasGitlab

		if !hasValue && !hasValueFrom {
			return fmt.Errorf("%s: must have value or valueFrom", prefix)
		}

		if hasValue && hasValueFrom {
			return fmt.Errorf("%s: cannot have both value and valueFrom", prefix)
		}

		if hasGCP && hasGitlab {
			return fmt.Errorf("%s: cannot have both gcpSecretKeyRef and gitlabVariableKeyRef", prefix)
		}

		if hasGitlab {
			if env.ValueFrom.GitlabVariableKeyRef.Project == "" {
				return fmt.Errorf("%s: gitlabVariableKeyRef.project is required", prefix)
			}
		}
	}

	return nil
}
