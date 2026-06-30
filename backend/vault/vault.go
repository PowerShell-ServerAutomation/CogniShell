package vault

import (
	"context"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
)

// Client defines our wrapper around the Vault API client
type Client struct {
	client *vault.Client
}

// NewClient initializes a Vault client using AppRole authentication
func NewClient(addr string, roleID, secretID string) (*Client, error) {
	config := vault.DefaultConfig()
	config.Address = addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize vault client: %w", err)
	}

	appRoleAuth, err := auth.NewAppRoleAuth(roleID, &auth.SecretID{FromString: secretID})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize approle auth: %w", err)
	}

	authInfo, err := client.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		return nil, fmt.Errorf("failed to login via approle: %w", err)
	}
	if authInfo == nil {
		return nil, fmt.Errorf("no auth info returned during login")
	}

	return &Client{client: client}, nil
}

// ReadSecrets fetches secrets from a KV-v2 path (e.g. "secret/data/cognishell/dev")
func (c *Client) ReadSecrets(ctx context.Context, fullPath string) (map[string]interface{}, error) {
	mountPath, secretPath := ParseVaultPath(fullPath)
	if mountPath == "" || secretPath == "" {
		return nil, fmt.Errorf("invalid vault path: %s", fullPath)
	}

	kv := c.client.KVv2(mountPath)
	secret, err := kv.Get(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from path %s: %w", fullPath, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no secrets found at path: %s", fullPath)
	}

	return secret.Data, nil
}

// ParseVaultPath splits a full Vault path into mount path and relative path
func ParseVaultPath(fullPath string) (string, string) {
	parts := strings.Split(strings.Trim(fullPath, "/"), "/")
	if len(parts) == 0 {
		return "", ""
	}
	mountPath := parts[0]
	
	startIdx := 1
	if len(parts) > 2 && parts[1] == "data" {
		startIdx = 2
	}
	
	relativePath := strings.Join(parts[startIdx:], "/")
	return mountPath, relativePath
}
