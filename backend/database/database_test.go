package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestDatabase_Integration(t *testing.T) {
	// Try loading environment variables from .env files
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		t.Skip("Skipping PostgreSQL integration test: missing credentials in env")
	}

	if dbPort == "" {
		dbPort = "5432"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
	client, err := NewClient(connStr)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Init Schema
	err = client.InitSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize database schema: %v", err)
	}

	// 2. Insert dummy record
	recordID := fmt.Sprintf("test-uuid-%d", time.Now().UnixNano())
	record := ExecutionRecord{
		ID:              recordID,
		ScriptName:      "test-script.ps1",
		Branch:          "main",
		Environment:     "test",
		ExitCode:        0,
		DurationSeconds: 1.25,
		ExecutedAt:      time.Now(),
		Error:           "",
	}

	err = client.InsertExecution(ctx, record)
	if err != nil {
		t.Fatalf("Failed to insert execution record: %v", err)
	}

	// 3. Purge records older than 1 second from now (so our record gets purged)
	// We can use a timestamp slightly in the future to make sure it purges
	cutoffTime := time.Now().Add(5 * time.Second)
	rowsDeleted, err := client.PurgeRecords(ctx, cutoffTime)
	if err != nil {
		t.Fatalf("Failed to purge records: %v", err)
	}

	if rowsDeleted < 1 {
		t.Errorf("Expected at least 1 row to be deleted, got %d", rowsDeleted)
	}
}
