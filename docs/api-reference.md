# API Reference: CogniShell

CogniShell exposes a JSON-based REST API for execution management, system health, and monitoring telemetry.

---

## 1. Base URL
In a default local development environment, the API is available at:
`http://localhost:8080`

---

## 2. API Endpoints

### Execute Script
Triggers the execution of a PowerShell script fetched dynamically from the configured GitHub repository.

* **Path**: `/api/v1/execute`
* **Method**: `POST`
* **Headers**:
  * `Content-Type: application/json`

#### Request Payload
| Field Name | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `script_path` | `string` | **Yes** | Relative path to the `.ps1` file within the GitHub repository (e.g., `scripts/restart-service.ps1`). |
| `branch` | `string` | No | Git branch to retrieve the script from. Defaults to the repository's default branch (usually `main`). |
| `owner` | `string` | No | GitHub organization or username. Defaults to the `GITHUB_REPO_OWNER` environment variable. |
| `repo` | `string` | No | GitHub repository name. Defaults to the `GITHUB_REPO_NAME` environment variable. |
| `environment`| `string` | No | The target execution environment (e.g., `dev`, `staging`, `prod`). Used to pull corresponding credential secrets from Vault. |

*Example Request Body:*
```json
{
  "script_path": "infrastructure/Clean-TempFiles.ps1",
  "branch": "release-v1",
  "owner": "PowerShell-ServerAutomation",
  "repo": "cognishell-scripts",
  "environment": "dev"
}
```

#### Success Response (200 OK)
Returned when the server successfully fetched, processed, and finished executing the script, regardless of whether the script execution succeeded or failed internally.

*Response Schema:*
* `execution_id` (`string`): Unique UUID tracking this runtime event.
* `stdout` (`string`): Full standard output stream returned from the PowerShell runspace.
* `stderr` (`string`): Full standard error stream returned from the execution.
* `exit_code` (`int`): Exit code returned by the `pwsh` process (0 = Success, >0 = Failure).
* `error` (`string`): Any internal system coordination error (omitted if null).

*Example Response (Script Success):*
```json
{
  "execution_id": "9a38f321-df62-4632-a50d-d7be30d074b1",
  "stdout": "Purging temporary folders...\nCleanup completed. 150MB freed.\n",
  "stderr": "",
  "exit_code": 0
}
```

*Example Response (Script Execution Error):*
```json
{
  "execution_id": "c10d32e9-74d1-432d-94c6-2c5e5fb78bf9",
  "stdout": "Initializing connection...\n",
  "stderr": "Connect-Server : Network connection timed out.\nAt line:3 char:1\n+ Connect-Server -Target \"10.0.0.12\"\n",
  "exit_code": 1
}
```

#### Error Response (400 Bad Request)
Returned when request validation fails (e.g., missing required fields, invalid JSON formats, or missing repository targets).

*Example Response:*
```json
{
  "execution_id": "2e0a2938-12ab-4bb3-8cc1-6fe78ba21aa9",
  "error": "Invalid request payload: script_path is required"
}
```

#### Error Response (500 Internal Server Error)
Returned when the server encounters a critical infrastructure error that blocks execution entirely (e.g., cannot connect to GitHub API, Vault secrets access denied, database failures, or host environment issues).

---

### Health Check
Retrieves the operational status of the Go server.

* **Path**: `/health`
* **Method**: `GET`

#### Success Response (200 OK)
*Example Response:*
```json
{
  "status": "healthy"
}
```

---

### Telemetry Metrics
Exposes Prometheus-formatted time-series metrics.

* **Path**: `/metrics`
* **Method**: `GET`

#### Response Format
Plain text payload conforming to the Prometheus scraping standard. 

#### Core Metrics Exposed
* `script_success_total`: Labeled by `script_path`. Tracks the count of successful script executions (ExitCode = 0).
* `script_failure_total`: Labeled by `script_path`. Tracks the count of failed executions (ExitCode > 0).
* `script_duration_seconds`: Histogram tracking the duration of script execution cycles from start of pwsh process to exit.

*Example Prometheus Metrics Output:*
```text
# HELP script_success_total Total number of successful script executions.
# TYPE script_success_total counter
script_success_total{script_path="Test-Connection.ps1"} 12
script_success_total{script_path="infrastructure/Clean-TempFiles.ps1"} 3

# HELP script_failure_total Total number of failed script executions.
# TYPE script_failure_total counter
script_failure_total{script_path="Test-Failure.ps1"} 2

# HELP script_duration_seconds Script execution duration in seconds.
# TYPE script_duration_seconds histogram
script_duration_seconds_bucket{script_path="Test-Connection.ps1",le="0.1"} 0
script_duration_seconds_bucket{script_path="Test-Connection.ps1",le="0.5"} 2
script_duration_seconds_bucket{script_path="Test-Connection.ps1",le="1"} 8
script_duration_seconds_bucket{script_path="Test-Connection.ps1",le="+Inf"} 12
script_duration_seconds_sum{script_path="Test-Connection.ps1"} 9.42
script_duration_seconds_count{script_path="Test-Connection.ps1"} 12
```
