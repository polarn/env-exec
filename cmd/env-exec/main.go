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

func main() {
	args := os.Args[1:]
	dryRun := false

	// Parse flags
	for len(args) > 0 {
		switch args[0] {
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

	envVars := make(map[string]string)
	for _, p := range provider.AllProviders() {
		if err := p.Provide(cfg, envVars); err != nil {
			log.Fatalf("Error: %v", err)
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
