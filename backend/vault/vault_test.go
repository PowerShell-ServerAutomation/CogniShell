package vault

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestVaultClient_Integration(t *testing.T) {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	addr := os.Getenv("VAULT_ADDR")
	roleID := os.Getenv("VAULT_ROLE_ID")
	secretID := os.Getenv("VAULT_SECRET_ID")
	secretPath := os.Getenv("VAULT_SECRET_PATH")

	if addr == "" || roleID == "" || secretID == "" || secretPath == "" {
		t.Skip("Skipping Vault integration test: missing credentials in env")
	}

	client, err := NewClient(addr, roleID, secretID)
	if err != nil {
		t.Fatalf("Failed to initialize Vault client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secrets, err := client.ReadSecrets(ctx, secretPath)
	if err != nil {
		t.Fatalf("Failed to read secrets: %v", err)
	}

	if secrets == nil {
		t.Fatal("Secrets map is nil")
	}

	if _, ok := secrets["TEST_SECRET_VALUE"]; !ok {
		t.Errorf("Expected 'TEST_SECRET_VALUE' key in secrets, but it was not found. Available keys: %v", secrets)
	}
}

func TestParseVaultPath(t *testing.T) {
	tests := []struct {
		input     string
		wantMount string
		wantPath  string
	}{
		{"secret/data/cognishell/dev", "secret", "cognishell/dev"},
		{"secret/cognishell/dev", "secret", "cognishell/dev"},
		{"custom_mount/data/some/nested/secret", "custom_mount", "some/nested/secret"},
		{"/secret/data/path/", "secret", "path"},
	}

	for _, tt := range tests {
		gotMount, gotPath := ParseVaultPath(tt.input)
		if gotMount != tt.wantMount || gotPath != tt.wantPath {
			t.Errorf("ParseVaultPath(%q) = (%q, %q), want (%q, %q)", tt.input, gotMount, gotPath, tt.wantMount, tt.wantPath)
		}
	}
}
