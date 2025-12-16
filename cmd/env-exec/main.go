package main

import (
	"log"
	"os"

	"github.com/polarn/env-exec/internal/config"
	"github.com/polarn/env-exec/internal/env"
	"github.com/polarn/env-exec/internal/exec"
	"github.com/polarn/env-exec/internal/provider"
)

func main() {
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

	if len(os.Args) < 2 {
		env.Print(envVars)
	} else {
		if err := env.Set(envVars); err != nil {
			log.Fatalf("Error: %v", err)
		}
		if err := exec.Run(os.Args[1:]); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}
