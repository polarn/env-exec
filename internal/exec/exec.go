package exec

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func Run(args []string) error {
	command := args[0]
	cmdArgs := args[1:]

	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		return fmt.Errorf("failed to run command: %w", err)
	}
	return nil
}
