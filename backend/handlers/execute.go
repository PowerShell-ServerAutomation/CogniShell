package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cognishell/database"
	"cognishell/executor"
	"cognishell/github"
	"cognishell/loki"
	"cognishell/metrics"
	"cognishell/notification"
	"cognishell/ollama"
	"cognishell/vault"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExecuteRequest defines the input payload for the execute endpoint
type ExecuteRequest struct {
	ScriptPath  string `json:"script_path" binding:"required"`
	Branch      string `json:"branch"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Environment string `json:"environment"`
}

// ExecuteResponse defines the output payload for the execute endpoint
type ExecuteResponse struct {
	ExecutionID string `json:"execution_id"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	ExitCode    int    `json:"exit_code"`
	Error       string `json:"error,omitempty"`
}

// ExecuteHandler handles execution HTTP requests
type ExecuteHandler struct {
	ghClient     *github.Client
	executor     executor.Executor
	vaultClient  *vault.Client
	dbClient     *database.Client
	lokiClient   *loki.Client
	ollamaClient *ollama.Client
	notifier     *notification.Client
}

// NewExecuteHandler creates a new ExecuteHandler
func NewExecuteHandler(
	ghClient *github.Client,
	exec executor.Executor,
	vaultClient *vault.Client,
	dbClient *database.Client,
	lokiClient *loki.Client,
	ollamaClient *ollama.Client,
	notifier *notification.Client,
) *ExecuteHandler {
	return &ExecuteHandler{
		ghClient:     ghClient,
		executor:     exec,
		vaultClient:  vaultClient,
		dbClient:     dbClient,
		lokiClient:   lokiClient,
		ollamaClient: ollamaClient,
		notifier:     notifier,
	}
}

// HandleExecute handles POST /api/v1/execute requests
func (h *ExecuteHandler) HandleExecute(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ExecuteResponse{
			Error: "Invalid request payload: script_path is required",
		})
		return
	}

	// Generate a unique Execution ID
	execID := uuid.New().String()

	// Resolve Owner and Repo from request or environment
	owner := req.Owner
	if owner == "" {
		owner = os.Getenv("GITHUB_REPO_OWNER")
	}
	repo := req.Repo
	if repo == "" {
		repo = os.Getenv("GITHUB_REPO_NAME")
	}

	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, ExecuteResponse{
			ExecutionID: execID,
			Error:       "Repository owner and name must be specified in the request or environment",
		})
		return
	}

	// Resolve Branch
	branch := req.Branch
	if branch == "" {
		branch = "main" // Default to main
	}

	// Setup execution context with a reasonable timeout (e.g., 30 seconds)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 1. Resolve target environment and fetch secrets from Vault
	envName := req.Environment
	if envName == "" {
		envName = os.Getenv("ENVIRONMENT")
		if envName == "" {
			envName = "dev"
		}
	}
	// Map "development" to "dev" for consistency if needed
	if envName == "development" {
		envName = "dev"
	}

	basePath := os.Getenv("VAULT_SECRET_PATH")
	var secretPath string
	if basePath != "" {
		if strings.HasSuffix(basePath, "/dev") {
			secretPath = strings.TrimSuffix(basePath, "/dev") + "/" + envName
		} else if strings.HasSuffix(basePath, "/development") {
			secretPath = strings.TrimSuffix(basePath, "/development") + "/" + envName
		} else {
			secretPath = basePath
		}
	} else {
		secretPath = "secret/data/cognishell/" + envName
	}

	envVars := make(map[string]string)
	if h.vaultClient != nil {
		secrets, err := h.vaultClient.ReadSecrets(ctx, secretPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ExecuteResponse{
				ExecutionID: execID,
				Error:       "Failed to fetch environment secrets from Vault: " + err.Error(),
			})
			return
		}
		for k, v := range secrets {
			if strVal, ok := v.(string); ok {
				envVars[k] = strVal
			} else {
				envVars[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	// 2. Fetch script from GitHub
	scriptContent, err := h.ghClient.FetchScript(ctx, owner, repo, req.ScriptPath, branch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ExecuteResponse{
			ExecutionID: execID,
			Error:       "Failed to fetch script from GitHub: " + err.Error(),
		})
		return
	}

	// 3. Execute script via PowerShell with injected environment variables
	startTime := time.Now()
	stdout, stderr, exitCode, err := h.executor.Execute(ctx, scriptContent, envVars)
	duration := time.Since(startTime).Seconds()

	// 4. Send logs to Loki asynchronously (or synchronously, but here we do it synchronously to ensure they are sent before responding)
	if h.lokiClient != nil {
		if lokiErr := h.lokiClient.PushLogs(ctx, req.ScriptPath, execID, envName, stdout, stderr); lokiErr != nil {
			fmt.Printf("Warning: failed to push logs to Loki: %v\n", lokiErr)
		}
	}

	// 5. Store execution record in PostgreSQL
	if h.dbClient != nil {
		var errMsg string
		if err != nil {
			errMsg = err.Error()
		}
		dbErr := h.dbClient.InsertExecution(ctx, database.ExecutionRecord{
			ID:              execID,
			ScriptName:      req.ScriptPath,
			Branch:          branch,
			Environment:     envName,
			ExitCode:        exitCode,
			DurationSeconds: duration,
			ExecutedAt:      startTime,
			Error:           errMsg,
		})
		if dbErr != nil {
			fmt.Printf("Warning: failed to write execution record to PostgreSQL: %v\n", dbErr)
		}
	}

	// 6. Record metrics
	if err != nil || exitCode != 0 {
		metrics.ScriptFailureTotal.WithLabelValues(req.ScriptPath, envName).Inc()
		if h.ollamaClient != nil {
			go h.triggerAutoRemediation(owner, repo, branch, envName, req.ScriptPath, scriptContent, stderr, exitCode)
		}
	} else {
		metrics.ScriptSuccessTotal.WithLabelValues(req.ScriptPath, envName).Inc()
	}
	metrics.ScriptDurationSeconds.WithLabelValues(req.ScriptPath, envName).Observe(duration)

	if err != nil {
		c.JSON(http.StatusInternalServerError, ExecuteResponse{
			ExecutionID: execID,
			Stdout:      stdout,
			Stderr:      stderr,
			ExitCode:    exitCode,
			Error:       "Execution failed: " + err.Error(),
		})
		return
	}

	// 7. Return output response
	c.JSON(http.StatusOK, ExecuteResponse{
		ExecutionID: execID,
		Stdout:      stdout,
		Stderr:      stderr,
		ExitCode:    exitCode,
	})
}

// triggerAutoRemediation handles background LLM analysis, issue assignment, and Slack/Teams alerts
func (h *ExecuteHandler) triggerAutoRemediation(owner, repo, branch, env, scriptPath, scriptContent, stderr string, exitCode int) {
	// Create context with timeout for background processes
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("[REMEDIATION] Fetching CODEOWNERS file for %s/%s...\n", owner, repo)
	codeownersContent, err := h.ghClient.FetchCODEOWNERS(ctx, owner, repo)
	var assignees []string
	if err != nil {
		fmt.Printf("[REMEDIATION] Warning: failed to fetch CODEOWNERS: %v\n", err)
	} else {
		assignees = github.ParseCODEOWNERS(codeownersContent, scriptPath)
		fmt.Printf("[REMEDIATION] Discovered assignees from CODEOWNERS: %v\n", assignees)
	}

	// 2. Query Ollama for error analysis
	fmt.Printf("[REMEDIATION] Querying Ollama LLM for analysis of %s...\n", scriptPath)
	analysis, err := h.ollamaClient.AnalyzeError(ctx, scriptPath, scriptContent, stderr)
	if err != nil {
		fmt.Printf("[REMEDIATION] Error: failed to get Ollama analysis: %v\n", err)
		analysis = fmt.Sprintf("Failed to get analysis from Ollama: %v\n\nOriginal Stderr:\n```\n%s\n```", err, stderr)
	}

	// 3. Create GitHub Issue
	title := fmt.Sprintf("Bug Analysis & Remediation: %s failed in %s", scriptPath, env)
	body := fmt.Sprintf(
		"### Script Execution Failure Alert\n\n"+
			"- **Script Path:** `%s`\n"+
			"- **Environment:** `%s`\n"+
			"- **Exit Code:** `%d`\n"+
			"- **Target Branch:** `%s`\n"+
			"- **Assignees:** %s\n\n"+
			"%s",
		scriptPath, env, exitCode, branch, strings.Join(assignees, ", "), analysis,
	)

	fmt.Printf("[REMEDIATION] Creating GitHub Issue for %s/%s...\n", owner, repo)
	issue, err := h.ghClient.CreateIssue(ctx, owner, repo, title, body, assignees)
	var issueURL string
	if err != nil {
		fmt.Printf("[REMEDIATION] Warning: failed to create GitHub issue with assignees: %v. Retrying without assignees...\n", err)
		// Retry without assignees (set to nil) and mention them in the body
		bodyWithMentions := fmt.Sprintf("%s\n\n---\n*Note: Assignees could not be assigned directly via the API. Mentions: %s*", body, strings.Join(assignees, ", "))
		issue, err = h.ghClient.CreateIssue(ctx, owner, repo, title, bodyWithMentions, nil)
		if err != nil {
			fmt.Printf("[REMEDIATION] Error: failed to create GitHub issue even without assignees: %v\n", err)
			issueURL = fmt.Sprintf("https://github.com/%s/%s/issues", owner, repo)
		} else {
			issueURL = issue.GetHTMLURL()
			fmt.Printf("[REMEDIATION] GitHub Issue successfully created (without assignees): %s\n", issueURL)
		}
	} else {
		issueURL = issue.GetHTMLURL()
		fmt.Printf("[REMEDIATION] GitHub Issue successfully created and assigned: %s\n", issueURL)
	}

	// 4. Send Slack/Teams Alert Webhook
	provider := os.Getenv("NOTIFICATION_PROVIDER")
	var webhookURL string
	if strings.ToLower(provider) == "teams" {
		webhookURL = os.Getenv("TEAMS_WEBHOOK_URL")
	} else if strings.ToLower(provider) == "slack" {
		webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	}

	if provider != "" && provider != "none" {
		fmt.Printf("[REMEDIATION] Sending alert via %s webhook...\n", provider)
		err = h.notifier.SendNotification(ctx, provider, webhookURL, scriptPath, env, assignees, issueURL)
		if err != nil {
			fmt.Printf("[REMEDIATION] Error: failed to send webhook notification: %v\n", err)
		} else {
			fmt.Printf("[REMEDIATION] Webhook notification sent successfully.\n")
		}
	}
}

