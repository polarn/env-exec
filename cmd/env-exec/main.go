package main

import (
	"fmt"
	"log"
	"os"

	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/env"
	"github.com/polarn/env-exec/internal/exec"
	"github.com/polarn/env-exec/internal/provider"
)

var (
	version = "dev"
	commit  = "none"
)

func printUsage() {
	fmt.Print(`Usage: env-exec [OPTIONS] [COMMAND] [ARGS...]

Inject environment variables from various sources before executing a command.

Options:
  -h, --help      Show this help message
  -v, --version   Show version information
  -n, --dry-run   Print environment variables without executing command

Environment:
  ENV_EXEC_YAML   Path to config file (default: .env-exec.yaml)
  GITLAB_TOKEN    GitLab API token (required for GitLab variables)

Examples:
  env-exec terraform plan       Run terraform with injected env vars
  env-exec --dry-run            Print env vars that would be set
  source <(env-exec)            Export env vars to current shell
`)
}

func main() {
	log.SetFlags(0)
	args := os.Args[1:]
	dryRun := false

	// Parse flags
	for len(args) > 0 {
		switch args[0] {
		case "--help", "-h":
			printUsage()
			return
		case "--version", "-v":
			fmt.Printf("env-exec %s (%s)\n", version, commit)
			return
		case "--dry-run", "-n":
			dryRun = true
			args = args[1:]
		default:
			goto done
		}
	}
done:

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if err := config.Validate(cfg); err != nil {
		log.Fatalf("Error: config: %v", err)
	}

	envVars := make(map[string]string)
	for _, p := range provider.AllProviders() {
		if err := p.Provide(cfg, envVars); err != nil {
			log.Fatalf("Error: %s: %v", p.Name(), err)
		}
	}

	if len(args) == 0 || dryRun {
		env.Print(envVars)
	} else {
		if err := env.Set(envVars); err != nil {
			log.Fatalf("Error: %v", err)
		}
		if err := exec.Run(args); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}
