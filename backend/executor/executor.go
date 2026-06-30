package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Executor defines script execution capability
type Executor interface {
	Execute(ctx context.Context, scriptContent string, envVars map[string]string) (string, string, int, error)
}

// PowerShellExecutor executes scripts in a PowerShell process
type PowerShellExecutor struct {
	shellCmd string
}

// NewPowerShellExecutor initializes the executor, detecting the shell command to use
func NewPowerShellExecutor() *PowerShellExecutor {
	// 1. Check if POWERSHELL_CMD env variable is set
	cmd := os.Getenv("POWERSHELL_CMD")
	if cmd != "" {
		return &PowerShellExecutor{shellCmd: cmd}
	}

	// 2. Otherwise, check if pwsh is in PATH
	if _, err := exec.LookPath("pwsh"); err == nil {
		return &PowerShellExecutor{shellCmd: "pwsh"}
	}

	// 3. Fallback to powershell
	return &PowerShellExecutor{shellCmd: "powershell"}
}

// GetShellCmd returns the detected shell command name
func (e *PowerShellExecutor) GetShellCmd() string {
	return e.shellCmd
}

// Execute runs the script by piping it to standard input of the shell command
func (e *PowerShellExecutor) Execute(ctx context.Context, scriptContent string, envVars map[string]string) (string, string, int, error) {
	// Setup command with NoProfile, NonInteractive, and reading command from stdin
	// We use the context to enforce timeouts/cancellation
	cmd := exec.CommandContext(ctx, e.shellCmd, "-NoProfile", "-NonInteractive", "-Command", "-")

	// Inject environment variables
	if len(envVars) > 0 {
		cmd.Env = os.Environ()
		for k, v := range envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Get pipes for stdout/stderr to capture output asynchronously
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Get stdin pipe
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", "", -1, fmt.Errorf("failed to start process: %w", err)
	}

	// Write script content to stdin in a separate goroutine
	errChan := make(chan error, 1)
	go func() {
		defer stdinPipe.Close()
		_, err := io.WriteString(stdinPipe, scriptContent)
		errChan <- err
	}()

	// Capture stdout and stderr
	stdoutChan := make(chan string, 1)
	stderrChan := make(chan string, 1)

	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, stdoutPipe)
		stdoutChan <- buf.String()
	}()

	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, stderrPipe)
		stderrChan <- buf.String()
	}()

	// Wait for stdin write to complete
	if err := <-errChan; err != nil {
		return "", "", -1, fmt.Errorf("failed to write script to stdin: %w", err)
	}

	// Wait for process to exit
	waitErr := cmd.Wait()

	// Retrieve outputs from channels
	stdout := <-stdoutChan
	stderr := <-stderrChan

	// Determine exit code
	exitCode := 0
	if waitErr != nil {
		if exitError, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return stdout, stderr, -1, fmt.Errorf("process wait failed: %w", waitErr)
		}
	}

	return stdout, stderr, exitCode, nil
}
