package exec

import (
	"context"
	"fmt"
	"os/exec"
)

// RunExecutableAsync starts the given service with the provided arguments in a non-blocking way.
func RunExecutableAsync(executablePath string, args []string, envVars map[string]string) (*exec.Cmd, error) {
	ctx := context.TODO()
	cmd := exec.CommandContext(ctx, executablePath, args...)

	// Add environment variables to the command.
	for k, v := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// RunExecutable starts the given service with the provided arguments in a blocking way.
func RunExecutable(executablePath string, args []string, envVars map[string]string) error {
	cmd, err := RunExecutableAsync(executablePath, args, envVars)
	if err != nil {
		return err
	}
	return cmd.Wait()
}
