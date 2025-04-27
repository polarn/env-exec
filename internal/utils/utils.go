package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/polarn/env-exec/internal/config"
	"gopkg.in/yaml.v3"
)

func LoadConfigFile() []byte {
	filename := ".env-exec.yaml"
	if os.Getenv("ENV_EXEC_YAML") != "" {
		filename = os.Getenv("ENV_EXEC_YAML")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// This is OK, means we will not expose any variables
			return nil
		} else if os.IsPermission(err) {
			log.Fatalf("Specific error: Permission denied to read file '%s'.\n", filename)
		} else {
			log.Fatalf("Specific error: An unexpected error occurred while reading file '%s'.\n", filename)
		}
		return nil
	}
	return data
}

func LoadConfig() *config.RootConfig {
	var config config.RootConfig

	data := LoadConfigFile()
	if data != nil {
		err := yaml.Unmarshal(data, &config)
		if err != nil {
			log.Fatalf("Error unmarshalling YAML: %v", err)
		}
	}
	return &config
}

func MakeProperEnvVarName(name string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_")
	properName := replacer.Replace(name)
	properName = strings.ToUpper(properName)
	return properName
}

func PrintEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		fmt.Printf("export %s=\"%s\"\n", key, value)
	}
}

func SetEnvVars(envVars map[string]string) {
	for key, value := range envVars {
		err := os.Setenv(key, value)
		if err != nil {
			fmt.Println("Error setting environment variable:", err)
			return
		}
	}
}

func CheckIfGCPSecretKeyRefExists(config *config.RootConfig) bool {
	for _, env := range config.Env {
		if env.ValueFrom.GCPSecretKeyRef.Name != "" {
			return true
		}
	}
	return false
}

func ExecuteCommand() error {
	command := os.Args[1]
	args := os.Args[2:]

	cmd := exec.Command(command, args...)

	// Set the standard input, output, and error streams to the current process's
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		// If the command exited with a non-zero status, the error will be of type *exec.ExitError
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				fmt.Printf("Command exited with status: %d\n", status.ExitStatus())
			} else {
				fmt.Printf("Command failed: %v\n", err)
			}
			os.Exit(1)
		} else {
			fmt.Printf("Failed to run command: %v\n", err)
			os.Exit(1)
		}
	}
	return cmd.Run()
}
