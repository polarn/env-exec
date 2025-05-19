package main

import (
	"os"

	"github.com/polarn/env-exec/internal/process"
	"github.com/polarn/env-exec/internal/utils"
)

func main() {
	var envVars = make(map[string]string)

	config := utils.LoadConfig()
	process.EnvVars(config, &envVars)
	process.EnvVarsGCP(config, &envVars)

	if len(os.Args) < 2 {
		utils.PrintEnvVars(envVars)
	} else {
		utils.SetEnvVars(envVars)
		utils.ExecuteCommand()
	}
}
