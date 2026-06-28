Development Roadmap: CogniShell-Server

This document breaks down the Phase 1 MVP requirements into actionable, sequential development sprints.

Sprint 1: Project Initialization & Infrastructure Base

Objective: Set up the local development environment and scaffolding for all components.

[x] Task 1.1: Initialize the Go backend repository (go mod init cognishell-server).

[x] Task 1.2: Initialize the Next.js frontend repository (npx create-next-app@latest cognishell-ui --typescript).

[ ] Task 1.3: Create a baseline docker-compose.yml for local development that includes:

HashiCorp Vault (Dev mode)

Prometheus

Grafana & Grafana Loki

PostgreSQL

Ollama (with Llama 3 or CodeLlama pulled)

[ ] Task 1.4: Verify all Docker containers start and communicate on the local Docker network.

Sprint 2: Core Execution Engine & GitHub Integration (Epic 1)

Objective: Build the Go REST API and the ability to run fetched PowerShell scripts.

[ ] Task 2.1: Set up a Go web framework (e.g., Gin, Fiber, or standard net/http) and define the /api/v1/execute POST endpoint.

[ ] Task 2.2: Integrate the go-github SDK to authenticate and dynamically fetch raw .ps1 file contents from a target GitHub repository branch into memory.

[ ] Task 2.3: Write the Go function to spawn a local pwsh (PowerShell Core) exec.Command.

[ ] Task 2.4: Pipe the downloaded script content into the pwsh process via standard input (stdin).

[ ] Task 2.5: Capture stdout, stderr, and the ExitCode from the pwsh process and return it in the API response.

Sprint 3: Security & Secrets (Epic 2)

Objective: Integrate Vault to ensure secure, credential-less script execution.

[ ] Task 3.1: Add the HashiCorp Vault Go SDK to the backend.

[ ] Task 3.2: Implement Vault authentication (e.g., AppRole) when the Go application starts.

[ ] Task 3.3: Create a helper function in Go to query a Vault KV path for the requested target environment's Functional ID credentials.

[ ] Task 3.4: Inject these credentials securely into the pwsh subprocess (e.g., via secure environment variables mapped only to that specific child process).

Sprint 4: Observability & Telemetry (Epic 3)

Objective: Ensure executions are monitored, logged, and visualized.

[ ] Task 4.1: Integrate the Prometheus Go client library. Register counters for script_success_total and script_failure_total, and a histogram for script_duration_seconds.

[ ] Task 4.2: Expose the /metrics endpoint in the Go application.

[ ] Task 4.3: Implement a logging client in Go to push the captured stdout/stderr from the pwsh run directly to the Grafana Loki API, tagging it with ScriptName and ExecutionID.

[ ] Task 4.4: Write a Go cron job (or simple ticker) to purge logs/Postgres entries older than 30 days.

Sprint 5: Developer UI / Portal (Epic 5)

Objective: Build the Next.js pane of glass for users.

[ ] Task 5.1: Design the main Next.js dashboard layout with a sidebar for the Script Catalog.

[ ] Task 5.2: Connect Next.js to the GitHub API to fetch and list scripts tagged as "Production Ready" from the main branch.

[ ] Task 5.3: Integrate a Markdown rendering library (e.g., react-markdown) to display the script's README.md when clicked.

[ ] Task 5.4: Embed Grafana panels (via iframe) into the Next.js UI for the selected script, passing the script name as a dashboard variable to show specific success/failure rates.

[ ] Task 5.5: Add a "Trigger Execution" button in the UI that calls the Go backend /api/v1/execute endpoint.

Sprint 6: AI-Driven Auto-Remediation (Epic 4)

Objective: Close the loop with automated, AI-assisted incident reporting.

[ ] Task 6.1: Update the Go execution logic: if ExitCode > 0, format a JSON payload containing the stderr string and the original script code.

[ ] Task 6.2: Create a Go client to send this payload to the local Ollama API endpoint with a strict prompt for bug analysis and remediation.

[ ] Task 6.3: Parse the Ollama text response and use the go-github SDK to open a new Issue in the target repository containing the AI analysis.

[ ] Task 6.4: Parse the CODEOWNERS file via the GitHub API to find the responsible developer for the failed script.

[ ] Task 6.5: Send an HTTP POST to a configured Slack/Teams webhook URL, alerting the team and tagging the assigned developer with a link to the new GitHub Issue.