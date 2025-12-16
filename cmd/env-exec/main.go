package main

import (
	"os"

	"github.com/polarn/env-exec/internal/provider"
	"github.com/polarn/env-exec/internal/utils"
)

func main() {
	envVars := make(map[string]string)
	config := utils.LoadConfig()

	for _, p := range provider.AllProviders() {
		p.Provide(config, envVars)
	}

	if len(os.Args) < 2 {
		utils.PrintEnvVars(envVars)
	} else {
		utils.SetEnvVars(envVars)
		utils.ExecuteCommand()
	}
}
