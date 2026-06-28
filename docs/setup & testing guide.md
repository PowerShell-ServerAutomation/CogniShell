Environment Setup and Testing Guide

This document outlines the local development environment configuration, networking considerations, and End-to-End (E2E) testing procedures for CogniShell-Server.

1. Networking Context (Windows + Multipass)

The application is being developed on a Windows 10 host, utilizing Multipass to run an Ubuntu VM hosting Docker containers.

The Challenge: The Go application running inside the Docker container (in the Multipass VM) cannot use localhost to reach services running natively on the Windows host (like Prometheus, Grafana, or Ollama).

The Solution: Use the Windows host's primary LAN IP address (e.g., 192.168.x.x) in the configuration file instead of localhost.
(Note: Run ipconfig in Windows PowerShell to find this IP).

2. Environment Configuration (.env)

The system relies on a .env file for configuration. Because we use HashiCorp Vault, the only secrets stored locally are the AppRole credentials required to authenticate with Vault.

Create a .env file in the root of your project:

# Server Config
APP_PORT=8080
ENVIRONMENT=development

# Infrastructure Endpoints (Use Windows LAN IP for Multipass compatibility)
PROMETHEUS_URL=http://<YOUR_WINDOWS_LAN_IP>:9090
LOKI_URL=http://<YOUR_WINDOWS_LAN_IP>:3100
OLLAMA_URL=http://<YOUR_WINDOWS_LAN_IP>:11434

# Vault Configuration
VAULT_ADDR=http://10.0.0.15:8200
VAULT_ROLE_ID=your-approle-role-id
VAULT_SECRET_ID=your-approle-secret-id
VAULT_SECRET_PATH=secret/data/cognishell/dev

# GitHub Configuration
GITHUB_TOKEN=your_personal_access_token # (Ideally fetched dynamically from Vault in later phases)
GITHUB_REPO_OWNER=your-username
GITHUB_REPO_NAME=cognishell-scripts


3. HashiCorp Vault Initialization

Before testing, Vault must be seeded with the necessary policies, AppRole, and target credentials. Run the following commands against your Vault instance (http://10.0.0.15:8200):

Enable AppRole Auth:

vault auth enable approle


Create a Policy:
Create a file named cognishell-policy.hcl:

path "secret/data/cognishell/*" {
  capabilities = ["read"]
}


Apply the policy:

vault policy write cognishell cognishell-policy.hcl


Create the AppRole:

vault write auth/approle/role/cognishell-server token_policies="cognishell" token_ttl=1h


Retrieve AppRole Credentials (Add these to your .env):

vault read auth/approle/role/cognishell-server/role-id
vault write -f auth/approle/role/cognishell-server/secret-id


Seed Mock Target Credentials:

vault kv put secret/cognishell/dev/target_server username="TestAdmin" password="SuperSecretPassword!"


4. Multipass Environment Preparation

To test the Docker container within the Ubuntu VM while coding on Windows, mount the project directory.

Mount the Directory (Run in Windows PowerShell):

# Replace with your actual paths
multipass mount C:\Path\To\CogniShell-Server primary:/home/ubuntu/cognishell


Access the VM:

multipass shell primary


Run the Application via Docker:

cd /home/ubuntu/cognishell
docker build -t cognishell-server .
docker run -p 8080:8080 --env-file .env cognishell-server


5. End-to-End (E2E) Testing Workflows

Once the environment is running, use these workflows to validate the core features.

Test Case 1: The "Happy Path" (Core Execution & Telemetry)

Commit a script named Test-Connection.ps1 to your GitHub repo that contains: Write-Output "Hello World"

Trigger the API from your Windows host (via Postman or cURL):

curl -X POST http://<MULTIPASS_VM_IP>:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{"script": "Test-Connection.ps1", "branch": "main", "target_env": "dev"}'


Validation:

API returns 200 OK with "Hello World" in the stdout JSON field.

Check Grafana: Verify Prometheus script_success_total incremented by 1.

Test Case 2: Dynamic Secret Injection

Commit a script Test-Auth.ps1 containing: Write-Output "Target User is $env:TARGET_USERNAME"

Trigger the API.

Validation:

The backend authenticates via Vault AppRole, retrieves the credentials, and securely injects them into the pwsh process.

API returns 200 OK with "Target User is TestAdmin".

Test Case 3: AI-Remediation & Incident Response (Failure Path)

Commit a script Force-Fail.ps1 containing a syntax error (e.g., Write-Hostt "This will fail").

Trigger the API.

Validation:

API returns a non-zero Exit Code.

Check Ollama logs: Verify the Go backend queried the local LLM.

Check GitHub: Verify a new Issue is created containing the LLM's root-cause analysis.

Check Slack/Teams (If configured): Verify the webhook notification arrived tagging the CODEOWNER.

Developer Tip: While Docker via Multipass mimics production, compiling and running Go natively on your Windows 10 desktop (go run main.go) during active development will significantly speed up iteration. The native Go binary will still successfully communicate with your Vault and Grafana endpoints.