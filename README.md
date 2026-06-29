# CogniShell: AI-Augmented Centralized PowerShell Automation

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.20%2B-blue)](backend/go.mod)
[![Next.js Version](https://img.shields.io/badge/Next.js-13%2B-black)](frontend/package.json)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

CogniShell is a secure, centralized, and AI-augmented coordination engine designed to modernize enterprise PowerShell automation. By bridging version control (GitHub), secrets management (HashiCorp Vault), telemetry (Prometheus & Grafana Loki), and local LLMs (Ollama), CogniShell transitions administrative scripting from scattered execution to a secure GitOps model.

---

## 1. Problem & Solution

| Traditional PowerShell Executions | The CogniShell Solution |
| :--- | :--- |
| **Scattered Execution**: Scripts run on local workstations or individual task schedulers with zero audit trail. | **Centralized Engine**: Run scripts via a stateless Go API controller or Next.js developer console. |
| **Credential Sprawl**: Secrets and passwords hardcoded directly in script bodies or text configuration files. | **Zero-Trust Injection**: Functional IDs fetched dynamically from HashiCorp Vault at runtime and injected in-memory. |
| **Silent Failures & Manual Triage**: Failed runs go unnoticed, requiring manual log digging to find the root cause. | **AI Self-Healing**: Local Ollama LLM analyzes failed stderr, opens GitHub issues, and alerts developers. |
| **No Consolidated Logs**: Standard output and errors are lost or scattered across local file systems. | **Stream Telemetry**: Logs streamed in real-time to Grafana Loki; metrics aggregated by Prometheus. |

---

## 2. Core Capabilities

* **GitOps Execution**: Fetch `.ps1` automation code dynamically from GitHub branches into memory at runtime. Scripts are never stored on execution host disks.
* **Zero-Trust secrets**: Backend authenticates via Vault AppRole at runtime, retrieves functional credentials, and injects them as secure environment variables scoped strictly to the child `pwsh` process.
* **AI Incident Diagnostics**: Script failures trigger the local Ollama LLM (e.g., CodeLlama). The LLM performs root-cause analysis, suggests a fix, opens a GitHub issue, and assigns the CODEOWNER.
* **Telemetry & ChatOps**: live-stream stdout/stderr logs to Grafana Loki, record execution counts and durations in Prometheus, and dispatch incident notifications to Slack/Teams channels.
* **Developer Web Console**: Next.js UI showcasing the version-controlled script catalog, rendering Markdown documentation, displaying success/failure stats, and providing execution triggers.

---

## 3. High-Level Architecture

```mermaid
flowchart TD
    %% Users and UI
    Dev([Developer / Webhook]) -->|Triggers Execution| UI[Next.js Portal]
    UI -->|API Call| Engine[Go Execution Engine]
    
    %% Engine Integrations
    Engine <-->|1. Fetch Script| GitHub[GitHub API]
    Engine <-->|2. Fetch Secrets| Vault[(HashiCorp Vault)]
    
    %% Execution
    Engine -->|3. Spawns| PWSH[pwsh Process]
    PWSH -->|Executes via SSH/WinRM| Target[Target Servers]
    
    %% Observability
    Engine -->|4. Push Logs| Loki[(Grafana Loki)]
    Engine -->|Expose Metrics| Prom[(Prometheus)]
    Prom --> Grafana[Grafana Dashboards]
    Loki --> Grafana
    Grafana -.->|Embeds Panels| UI
    
    %% AI Auto-Remediation (Failure Path)
    Engine -.->|5. On Error: Send Context| Ollama{Ollama LLM}
    Ollama -.->|Analysis| Engine
    Engine -.->|6. Create Issue| GitHub
    Engine -.->|7. Alert CODEOWNER| ChatOps[Slack / Teams]
    
    %% Styling
    classDef primary fill:#2b3137,stroke:#fff,stroke-width:2px,color:#fff;
    classDef secondary fill:#0052cc,stroke:#fff,stroke-width:2px,color:#fff;
    classDef db fill:#00ADD8,stroke:#fff,stroke-width:2px,color:#fff;
    classDef alert fill:#e34f26,stroke:#fff,stroke-width:2px,color:#fff;
    
    class Engine,UI primary;
    class GitHub,Vault,Target secondary;
    class Loki,Prom,Postgres db;
    class Ollama,ChatOps alert;
```

---

## 4. Quick Start

Get a local development cluster running in under 5 minutes.

### 1. Spin Up Infrastructure
Run the containerized Vault, Postgres, Prometheus, Loki, Grafana, and Ollama services:
```bash
docker compose up -d
```

### 2. Configure Your Environment
Create your `.env` configuration file from the template:
```bash
cp .env.example .env
```
Fill out the variables in `.env` (including GitHub tokens and Vault AppRole credentials. See the [Developer Guide](docs/developer-guide.md) for Vault initialization commands).

### 3. Run the Go Backend
```bash
cd backend
go run main.go
```

### 4. Run the Next.js Frontend Console
```bash
cd frontend
npm install
npm run dev
```
Navigate to `http://localhost:3001` in your browser.

### 5. Trigger an Execution (Example)
```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{"script_path": "Test-HappyPath.ps1", "environment": "dev"}'
```

---

## 5. Detailed Documentation Directories

To learn more about setting up, scaling, and using the platform, refer to the guides below:

* 📘 **[Developer Guide](docs/developer-guide.md)**: Deep dive into codebase setup, architecture details, Vault seeding, and local networking.
* 🧪 **[Testing Guide](docs/testing.md)**: Details unit testing, linting, and step-by-step manual E2E validation scenarios.
* ⚙️ **[Deployment Guide](docs/deployment.md)**: Details standard Docker deployment, environment configuration checklists, and our **Enterprise HA scaling roadmap** (task queues, ephemeral Job executors, cloud database clustering).
* 🔌 **[API Reference](docs/api-reference.md)**: Reference manual for REST API payloads, JSON structures, error response schemas, and Prometheus telemetry metrics.

For community contribution, standards, and future plans:
* 🗺️ **[Product Roadmap](ROADMAP.md)**: Future plans, security hardening, and feature backlog for Phase 2 (MVP2).
* 🤝 **[Contributing Guide](CONTRIBUTING.md)**: Pull Request processes, branch naming formats, and coding guidelines.
* 📜 **[Code of Conduct](CODE_OF_CONDUCT.md)**: General standards of conduct for our developers and maintainers.
