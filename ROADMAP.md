# CogniShell Product Roadmap: MVP2 & Beyond

This document outlines the proposed roadmap, architecture improvements, and feature expansions for **Phase 2 (MVP2)** of CogniShell, focusing on security hardening, workflow scalability, and enhanced AI-driven operations.

---

## 1. Core Focus Areas for MVP2

While MVP1 established the functional local prototype, **MVP2** transitions CogniShell into a secure, multi-tenant, and enterprise-grade automation platform.

```
                  ┌──────────────────────────────────────────┐
                  │          CogniShell UI (SSO/RBAC)        │
                  └────────────────────┬─────────────────────┘
                                       │ Interactive Trigger / Live Logs
                  ┌────────────────────▼─────────────────────┘
                  │          API Gateway (Load-Balanced)     │
                  └────────────────────┬─────────────────────┘
                                       │ Publish Job Message
                  ┌────────────────────▼─────────────────────┘
                  │       Distributed Queue (RabbitMQ/Redis) │
                  └────────────────────┬─────────────────────┘
                                       │ Poll Job
                  ┌────────────────────▼─────────────────────┘
                  │     Stateless Worker Pool (Horizontal)   │
                  └────────────────────┬─────────────────────┘
                                       │ Provision Ephemeral Runner
                  ┌────────────────────▼─────────────────────┘
                  │     Ephemeral Sandbox Pod (Kubernetes)   │
                  └──────────────────────────────────────────┘
```

---

## 2. Recommended MVP2 Feature Backlog

### 🔌 1. Distributed Workflows & Worker Queue (Scaling)
* **Goal**: Decouple script execution from the API Gateway to prevent CPU exhaustion during high-concurrency runs.
* **Proposed Implementation**:
  * Integrate a message broker (e.g., **RabbitMQ** or **Redis-backed Asynq**).
  * API Gateway posts jobs to the broker and returns an execution ID immediately (`202 Accepted`).
  * Dedicated Go Workers poll the broker and scale horizontally based on queue depth.

### 🔒 2. Ephemeral Sandbox Execution (Security Isolation)
* **Goal**: Prevent scripts from accessing the worker container's filesystem or environment variables (including Vault configurations).
* **Proposed Implementation**:
  * Upon claiming a task, the Worker provisions a short-lived **Kubernetes Job Pod** or **serverless container (AWS Fargate)** running a secure, hardened `pwsh` image.
  * Secrets are injected only into the transient container. The container is garbage-collected immediately upon exit.

### 💻 3. Interactive Web Execution & Live Logs (User Experience)
* **Goal**: Allow operators to execute scripts with custom input variables and view outputs in real-time.
* **Proposed Implementation**:
  * UI renders dynamic forms based on PowerShell script parameters (parsing `#Requires` or `.PARAMETER` comment blocks).
  * Stream stdout and stderr live to the Next.js UI using **WebSockets** or **Server-Sent Events (SSE)**.

### 🔑 4. Enterprise Access Control & Audit Compliance (Security)
* **Goal**: Standardize user roles, authenticate users, and track who executes what.
* **Proposed Implementation**:
  * Integrate **OAuth2 / OIDC** (OpenID Connect) for Single Sign-On (SSO) with providers like Okta, Azure AD, or Keycloak.
  * Implement **Role-Based Access Control (RBAC)** to restrict who can trigger scripts in `production` environments versus `dev` environments.
  * Record immutable audit logs in PostgreSQL tracking: User, Script, Environment, Arguments, Timestamp, and Hash of the script executed.

### 🖥️ 5. Remote Target Execution (WinRM & SSH)
* **Goal**: Execute scripts *remotely* on targeted Windows/Linux servers instead of locally inside the container space.
* **Proposed Implementation**:
  * Leverage Go-native WinRM or SSH libraries to establish connections.
  * Inject target operational credentials from Vault dynamically to establish the secure session.

### 🧠 6. Closed-Loop AI Remediation (AI Enhancement)
* **Goal**: Move from simple incident reporting to automated healing suggestions.
* **Proposed Implementation**:
  * Allow Ollama to automatically generate a **Git Pull Request (PR)** with the corrected script syntax to fix the bug directly in the GitHub repository.
  * Provide an "Auto-Retry with Remediation" button in the Web console for quick operator verification.

---

## 3. How to Publish Future Plans

To share and align these plans with stakeholders, you should utilize these publication paths:

1. **`ROADMAP.md` (This File)**: Maintained in the root of the repository. It gives immediate visibility to any developer clone.
2. **GitHub Milestones & Projects**: 
   * Create a milestone in GitHub titled `MVP2 (Enterprise Release)`.
   * Create issues corresponding to the stories above and assign them to the milestone.
   * Use GitHub Projects to visualize the sprint board (Backlog, In Progress, Review, Done).
3. **README Linking**: Link the `ROADMAP.md` in the main `README.md` under the "Detailed Documentation" section to ensure it remains a first-class citizen of your repository's landing page.
