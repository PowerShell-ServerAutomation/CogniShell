package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load env from parent directories
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = "http://10.0.0.11:8200"
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" || strings.HasPrefix(githubToken, "your_") {
		log.Println("Warning: GITHUB_TOKEN is not configured or is a placeholder. Please check your .env file.")
	}

	// 2. Create Vault client with root token
	config := vault.DefaultConfig()
	config.Address = vaultAddr
	client, err := vault.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}
	client.SetToken("myroot") // Dev mode root token

	// 3. Write policy
	policyName := "cognishell-policy"
	policyHcl := `
path "secret/data/cognishell/*" {
  capabilities = ["read", "list"]
}
`
	err = client.Sys().PutPolicy(policyName, policyHcl)
	if err != nil {
		log.Fatalf("Failed to put Vault policy: %v", err)
	}
	fmt.Printf("Created Vault policy: %s\n", policyName)

	// 4. Create AppRole role
	_, err = client.Logical().Write("auth/approle/role/cognishell-role", map[string]interface{}{
		"secret_id_ttl":  "0",
		"token_num_uses": "0",
		"token_ttl":      "20m",
		"token_max_ttl":  "30m",
		"token_policies": policyName,
	})
	if err != nil {
		log.Fatalf("Failed to create Vault AppRole role: %v", err)
	}
	fmt.Println("Created Vault AppRole: cognishell-role")

	// 5. Read Role ID
	secret, err := client.Logical().Read("auth/approle/role/cognishell-role/role-id")
	if err != nil || secret == nil {
		log.Fatalf("Failed to read Role ID: %v", err)
	}
	roleID := secret.Data["role_id"].(string)
	fmt.Printf("Role ID: %s\n", roleID)

	// 6. Generate Secret ID
	secret, err = client.Logical().Write("auth/approle/role/cognishell-role/secret-id", nil)
	if err != nil || secret == nil {
		log.Fatalf("Failed to generate Secret ID: %v", err)
	}
	secretID := secret.Data["secret_id"].(string)
	fmt.Printf("Secret ID: %s\n", secretID)

	// 7. Write secrets to KV store
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	kvData := map[string]interface{}{
		"GITHUB_TOKEN":       githubToken,
		"TEST_SECRET_VALUE": "VaultSecret12345!",
	}
	_, err = client.KVv2("secret").Put(ctx, "cognishell/dev", kvData)
	if err != nil {
		log.Fatalf("Failed to write secrets to Vault KV store: %v", err)
	}
	fmt.Println("Wrote secrets to Vault path: secret/data/cognishell/dev")

	// 8. Automatically update the .env file with the Role ID and Secret ID
	envFilePath := findEnvFile()
	if envFilePath != "" {
		err = updateEnvFile(envFilePath, roleID, secretID)
		if err != nil {
			log.Printf("Warning: Failed to update .env file: %v\n", err)
		} else {
			fmt.Printf("Successfully updated .env file at %s\n", envFilePath)
		}
	} else {
		fmt.Println("Could not locate .env file to update automatically.")
	}
}

func findEnvFile() string {
	paths := []string{".env", "../.env", "../../.env"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func updateEnvFile(path, roleID, secretID string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	hasRoleID := false
	hasSecretID := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "VAULT_ROLE_ID=") {
			lines[i] = fmt.Sprintf("VAULT_ROLE_ID=%s", roleID)
			hasRoleID = true
		} else if strings.HasPrefix(trimmed, "VAULT_SECRET_ID=") {
			lines[i] = fmt.Sprintf("VAULT_SECRET_ID=%s", secretID)
			hasSecretID = true
		}
	}

	newContent := strings.Join(lines, "\n")
	if !hasRoleID {
		newContent += fmt.Sprintf("\nVAULT_ROLE_ID=%s", roleID)
	}
	if !hasSecretID {
		newContent += fmt.Sprintf("\nVAULT_SECRET_ID=%s", secretID)
	}

	// Clean up carriage returns/newlines if any duplicate formatting is present
	newContent = strings.ReplaceAll(newContent, "\r\n", "\n")
	newContent = strings.ReplaceAll(newContent, "\n\n\n", "\n\n")

	return os.WriteFile(path, []byte(newContent), 0644)
}
