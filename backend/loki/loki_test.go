package loki

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestLokiClient_Integration(t *testing.T) {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		t.Skip("Skipping Loki integration test: LOKI_URL is not set")
	}

	client := NewClient(lokiURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Push logs
	err := client.PushLogs(ctx, "test-script.ps1", "test-exec-123", "test", "Hello stdout log line\nAnother stdout line", "Oops error line")
	if err != nil {
		t.Fatalf("Failed to push logs to Loki: %v", err)
	}

	// 2. Delete logs (Loki API test)
	// We delete logs matching the test environment
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(-5 * time.Second)
	err = client.DeleteLogs(ctx, start, end)
	if err != nil {
		t.Fatalf("Failed to delete logs: %v", err)
	}
}
