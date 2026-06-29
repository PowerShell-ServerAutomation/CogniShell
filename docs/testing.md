# Testing Guide: CogniShell

This guide describes the testing strategy, automated test commands, and manual End-to-End (E2E) verification workflows for CogniShell.

---

## 1. Automated Testing

### Backend (Go) Tests
The Go backend contains unit and integration tests for key components like GitHub integration, database logs, and secret extraction.

To execute the backend test suite:
1. Navigate to the `backend/` directory:
   ```bash
   cd backend
   ```
2. Run all tests:
   ```bash
   go test -v ./...
   ```
3. Run tests with coverage:
   ```bash
   go test -cover ./...
   ```

### Frontend (Next.js) Tests & Linting
Verify TypeScript static analysis and style criteria:
1. Navigate to the `frontend/` directory:
   ```bash
   cd frontend
   ```
2. Run the linter:
   ```bash
   npm run lint
   ```

---

## 2. End-to-End (E2E) Manual Verification Workflows

The E2E workflow validates the integration between the Go Backend Engine, HashiCorp Vault, local PowerShell runspace, Loki log streams, Prometheus telemetry metrics, and the Ollama LLM failure handler.

### Prerequisites for E2E Testing
1. Local infrastructure containers must be running (`docker compose up -d`).
2. HashiCorp Vault must be initialized and seeded with target secrets (see [developer-guide.md](developer-guide.md)).
3. The Go backend and Next.js frontend must be running.
4. You need a dedicated GitHub repository (e.g., `cognishell-scripts`) to hold test scripts, and your local `.env` must point to it.

---

### Test Case 1: The "Happy Path" (Core Execution & Telemetry)

This test confirms that CogniShell can fetch a PowerShell script from GitHub, run it locally, capture output, and record successful telemetry.

#### Steps:
1. In your GitHub script repository, commit a file named `Test-HappyPath.ps1` with the following content:
   ```powershell
   Write-Output "Hello from CogniShell Automation!"
   Get-Date
   ```
2. Trigger the execution API from your terminal using `cURL`:
   ```bash
   curl -X POST http://localhost:8080/api/v1/execute \
     -H "Content-Type: application/json" \
     -d '{
       "script_path": "Test-HappyPath.ps1",
       "branch": "main",
       "environment": "dev"
     }'
   ```
3. **Verify API Response**:
   Confirm the API returns `200 OK` with a JSON payload resembling:
   ```json
   {
     "execution_id": "4b7b3e0c-d92b-4223-a15d-85ad2de9b152",
     "stdout": "Hello from CogniShell Automation!\nTuesday, June 30, 2026 2:30:00 AM\n",
     "stderr": "",
     "exit_code": 0
   }
   ```
4. **Verify Telemetry Metrics**:
   * Open Prometheus at `http://localhost:9090`.
   * Search for the metric `script_success_total`.
   * Confirm the counter has incremented by `1` for the label `script_path="Test-HappyPath.ps1"`.
5. **Verify Stream Logs in Loki/Grafana**:
   * Open Grafana at `http://localhost:3000`.
   * Navigate to the **Explore** tab and select the **Loki** datasource.
   * Query the logs using LogQL: `{script_path="Test-HappyPath.ps1"}`.
   * Verify the stdout log line "Hello from CogniShell Automation!" is indexed and visible.

---

### Test Case 2: Dynamic Secret Injection

This test verifies that CogniShell securely retrieves credentials from HashiCorp Vault and injects them into the PowerShell runtime environment without exposing secrets in logs.

#### Steps:
1. Ensure Vault has a secret seeded at `secret/data/cognishell/dev` containing key/value: `username="TestAdmin"`.
2. Commit a script named `Test-Secrets.ps1` to your GitHub repo:
   ```powershell
   # Attempt to access environment variables injected by the engine
   $envUser = $env:TARGET_USERNAME
   if ($envUser) {
       Write-Output "Successfully authenticated! Connected as user: $envUser"
   } else {
       Write-Error "Failed: Secret environment variable TARGET_USERNAME not set."
       exit 1
   }
   ```
3. Trigger the execution API using the `dev` environment parameter:
   ```bash
   curl -X POST http://localhost:8080/api/v1/execute \
     -H "Content-Type: application/json" \
     -d '{
       "script_path": "Test-Secrets.ps1",
       "branch": "main",
       "environment": "dev"
     }'
   ```
4. **Verify API Response**:
   Confirm the API returns `200 OK` with:
   ```json
   {
     "execution_id": "84af4d21-f09c-49ee-8fa1-7ee8cb4c6020",
     "stdout": "Successfully authenticated! Connected as user: TestAdmin\n",
     "stderr": "",
     "exit_code": 0
   }
   ```
5. **Verify Logs Security**:
   * Query Loki logs in Grafana.
   * Verify that only `Successfully authenticated! Connected as user: TestAdmin` was written to the stream, and no operational passwords or Vault credentials were logged.

---

### Test Case 3: AI-Remediation & Incident Response (Failure Path)

This test validates that when a PowerShell script fails (non-zero exit code), CogniShell captures the error, queries the local Ollama LLM for analysis, logs the incident in GitHub Issues, and fires a ChatOps notification.

#### Steps:
1. Commit a faulty script named `Test-Failure.ps1` containing a deliberate syntax error or command failure:
   ```powershell
   Write-Output "Initiating database backup..."
   # Intentionally misspelling cmdlet to trigger a CommandNotFoundException
   Backup-SqlDatabasee -Database "MainDB" -BackupFile "C:\Backup.bak"
   ```
2. Trigger the execution API:
   ```bash
   curl -X POST http://localhost:8080/api/v1/execute \
     -H "Content-Type: application/json" \
     -d '{
       "script_path": "Test-Failure.ps1",
       "branch": "main",
       "environment": "dev"
     }'
   ```
3. **Verify API Response**:
   Confirm the API returns a response with `exit_code` matching the command failure (typically `1` or higher) and the `stderr` stream containing the PowerShell stack trace:
   ```json
   {
     "execution_id": "c10d32e9-74d1-432d-94c6-2c5e5fb78bf9",
     "stdout": "Initiating database backup...\n",
     "stderr": "Backup-SqlDatabasee : The term 'Backup-SqlDatabasee' is not recognized as the name of a cmdlet...\n",
     "exit_code": 1
   }
   ```
4. **Verify Ollama LLM Query**:
   * Inspect the Go backend console output.
   * Verify that the logs indicate: `Sending failure payload to Ollama for triage...` followed by `Ollama successfully returned analysis.`
5. **Verify GitHub Issue Creation**:
   * Open your GitHub repository on `github.com`.
   * Navigate to the **Issues** tab.
   * Check for a newly created issue titled something like: `[Incident] Execution Failure: Test-Failure.ps1 (Execution ID: c10d32e9-...)`.
   * Verify that the issue description contains:
     * The raw `stderr` dump.
     * The LLM's **Root Cause Analysis** explaining that `Backup-SqlDatabasee` has a typo (extraneous 'e' at the end).
     * The LLM's **Remediation Steps** suggesting the correct cmdlet `Backup-SqlDatabase` or installing the SQL Server module.
     * The issue is assigned to the developer mapped in `CODEOWNERS`.
6. **Verify ChatOps Notification**:
   * Check your Slack channel or Microsoft Teams channel (whichever was configured in `.env`).
   * Verify that a webhook alert card was posted containing:
     * The failed script name.
     * The assigned developer.
     * A direct link to the newly created GitHub Issue.
