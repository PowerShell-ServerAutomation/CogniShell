package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPowerShellExecutor_Execute_Success(t *testing.T) {
	exec := NewPowerShellExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := `Write-Output "Hello World from Test"`
	stdout, stderr, exitCode, err := exec.Execute(ctx, script, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	cleanedStdout := strings.TrimSpace(stdout)
	if cleanedStdout != "Hello World from Test" {
		t.Errorf("Expected stdout 'Hello World from Test', got '%s'", cleanedStdout)
	}

	if strings.TrimSpace(stderr) != "" {
		t.Errorf("Expected empty stderr, got '%s'", stderr)
	}
}

func TestPowerShellExecutor_Execute_Failure(t *testing.T) {
	exec := NewPowerShellExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := `Write-Error "An error occurred"; exit 12`
	_, stderr, exitCode, err := exec.Execute(ctx, script, nil)
	if err != nil {
		t.Fatalf("Expected no error from Execute wrapper itself, got: %v", err)
	}

	if exitCode != 12 {
		t.Errorf("Expected exit code 12, got %d", exitCode)
	}

	if !strings.Contains(stderr, "An error occurred") {
		t.Errorf("Expected stderr to contain 'An error occurred', got '%s'", stderr)
	}
}

func TestPowerShellExecutor_Execute_EnvInjection(t *testing.T) {
	exec := NewPowerShellExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := `Write-Output $env:MY_SECRET_ENV_VAR`
	envVars := map[string]string{
		"MY_SECRET_ENV_VAR": "Secret123Value!",
	}
	stdout, _, exitCode, err := exec.Execute(ctx, script, envVars)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	cleanedStdout := strings.TrimSpace(stdout)
	if cleanedStdout != "Secret123Value!" {
		t.Errorf("Expected stdout 'Secret123Value!', got '%s'", cleanedStdout)
	}
}
