package main

import (
	"log"
	"os"

	"github.com/polarn/env-exec/internal/provider"
	"github.com/polarn/env-exec/internal/utils"
)

func main() {
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	envVars := make(map[string]string)
	for _, p := range provider.AllProviders() {
		if err := p.Provide(config, envVars); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}

	if len(os.Args) < 2 {
		utils.PrintEnvVars(envVars)
	} else {
		if err := utils.SetEnvVars(envVars); err != nil {
			log.Fatalf("Error: %v", err)
		}
		if err := utils.ExecuteCommand(); err != nil {
			log.Fatalf("Error: %v", err)
		}
	}
}
