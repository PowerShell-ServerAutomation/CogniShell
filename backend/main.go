package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cognishell/database"
	"cognishell/executor"
	"cognishell/github"
	"cognishell/handlers"
	"cognishell/loki"
	_ "cognishell/metrics"
	"cognishell/notification"
	"cognishell/ollama"
	"cognishell/vault"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	// Load env file. Try current directory first, then parent directory.
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("Warning: No .env file found. Falling back to system environment variables.")
		}
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	// 1. Initialize Vault Client if configured
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultRoleID := os.Getenv("VAULT_ROLE_ID")
	vaultSecretID := os.Getenv("VAULT_SECRET_ID")
	vaultPath := os.Getenv("VAULT_SECRET_PATH")

	var vaultClient *vault.Client
	var err error
	var githubToken string

	if vaultAddr != "" && vaultRoleID != "" && vaultSecretID != "" {
		log.Printf("Connecting to HashiCorp Vault at %s...\n", vaultAddr)
		vaultClient, err = vault.NewClient(vaultAddr, vaultRoleID, vaultSecretID)
		if err != nil {
			log.Fatalf("Fatal: Failed to connect/authenticate with HashiCorp Vault: %v\n", err)
		}
		log.Println("Successfully authenticated with Vault via AppRole!")

		// Fetch GITHUB_TOKEN dynamically from Vault
		if vaultPath == "" {
			vaultPath = "secret/data/cognishell/dev"
		}
		log.Printf("Fetching GITHUB_TOKEN dynamically from Vault path: %s\n", vaultPath)
		secrets, err := vaultClient.ReadSecrets(context.Background(), vaultPath)
		if err != nil {
			log.Printf("Warning: Failed to fetch secrets from Vault path %s: %v. Falling back to environment GITHUB_TOKEN.\n", vaultPath, err)
			githubToken = os.Getenv("GITHUB_TOKEN")
		} else {
			if tok, ok := secrets["GITHUB_TOKEN"].(string); ok && tok != "" {
				githubToken = tok
				log.Println("Successfully retrieved GITHUB_TOKEN from Vault!")
			} else {
				log.Println("Warning: GITHUB_TOKEN not found or empty in Vault secret. Falling back to environment GITHUB_TOKEN.")
				githubToken = os.Getenv("GITHUB_TOKEN")
			}
		}
	} else {
		log.Println("Warning: Vault configuration missing. Running without Vault.")
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	// 1.5. Initialize PostgreSQL
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	var dbClient *database.Client
	if dbHost != "" && dbUser != "" && dbPassword != "" && dbName != "" {
		if dbPort == "" {
			dbPort = "5432"
		}
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)
		log.Printf("Connecting to PostgreSQL database at %s:%s...\n", dbHost, dbPort)

		dbClient, err = database.NewClient(connStr)
		if err != nil {
			log.Printf("Warning: Failed to connect to PostgreSQL: %v. Running without DB persistence.\n", err)
		} else {
			log.Println("Successfully connected to PostgreSQL database!")
			defer dbClient.Close()

			// Initialize DB schema
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := dbClient.InitSchema(ctx); err != nil {
				log.Fatalf("Fatal: Failed to initialize PostgreSQL database schema: %v\n", err)
			}
			log.Println("Database schema successfully initialized/verified.")
		}
	} else {
		log.Println("Warning: PostgreSQL configuration missing. Running without DB persistence.")
	}

	// 1.6. Initialize Loki client
	lokiURL := os.Getenv("LOKI_URL")
	var lokiClient *loki.Client
	if lokiURL != "" {
		log.Printf("Initializing Grafana Loki client at %s...\n", lokiURL)
		lokiClient = loki.NewClient(lokiURL)
	} else {
		log.Println("Warning: LOKI_URL missing. Running without log streaming to Loki.")
	}

	// 1.7. Start Background Purge Worker
	if dbClient != nil && lokiClient != nil {
		purgeIntervalStr := os.Getenv("PURGE_INTERVAL")
		purgeInterval := 24 * time.Hour
		if purgeIntervalStr != "" {
			if duration, err := time.ParseDuration(purgeIntervalStr); err == nil {
				purgeInterval = duration
			}
		}

		retentionPeriodStr := os.Getenv("RETENTION_PERIOD")
		retentionPeriod := 30 * 24 * time.Hour
		if retentionPeriodStr != "" {
			if duration, err := time.ParseDuration(retentionPeriodStr); err == nil {
				retentionPeriod = duration
			}
		}

		ctx, cancelPurge := context.WithCancel(context.Background())
		defer cancelPurge()

		go startPurgeWorker(ctx, dbClient, lokiClient, purgeInterval, retentionPeriod)
	}

	// Validate notification settings on startup
	notificationProvider := os.Getenv("NOTIFICATION_PROVIDER")
	teamsWebhook := os.Getenv("TEAMS_WEBHOOK_URL")
	slackWebhook := os.Getenv("SLACK_WEBHOOK_URL")

	if notificationProvider != "" && notificationProvider != "none" {
		prov := strings.ToLower(strings.TrimSpace(notificationProvider))
		if prov == "teams" && teamsWebhook == "" {
			log.Fatalf("Fatal: TEAMS_WEBHOOK_URL is mandatory when NOTIFICATION_PROVIDER is set to 'teams'")
		}
		if prov == "slack" && slackWebhook == "" {
			log.Fatalf("Fatal: SLACK_WEBHOOK_URL is mandatory when NOTIFICATION_PROVIDER is set to 'slack'")
		}
		if prov != "teams" && prov != "slack" {
			log.Fatalf("Fatal: Unsupported NOTIFICATION_PROVIDER: %s. Supported values: 'teams', 'slack', 'none'", notificationProvider)
		}
	}

	// Initialize Ollama client
	ollamaURL := os.Getenv("OLLAMA_URL")
	var ollamaClient *ollama.Client
	if ollamaURL != "" {
		log.Printf("Initializing Ollama client at %s...\n", ollamaURL)
		ollamaClient = ollama.NewClient(ollamaURL)
	} else {
		log.Println("Warning: OLLAMA_URL missing. Running without auto-remediation analysis.")
	}

	// Initialize notification client
	notifier := notification.NewClient()

	// 2. Initialize Components
	ghClient := github.NewClient(githubToken)
	powershellExecutor := executor.NewPowerShellExecutor()

	log.Printf("Initializing Executor using: %s\n", powershellExecutor.GetShellCmd())

	executeHandler := handlers.NewExecuteHandler(ghClient, powershellExecutor, vaultClient, dbClient, lokiClient, ollamaClient, notifier)

	// 3. Set up router
	r := gin.Default()
	r.Use(CORSMiddleware())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Execute endpoint
	r.POST("/api/v1/execute", executeHandler.HandleExecute)

	// Mock Webhook Receiver endpoint for local testing
	r.POST("/mock-webhook", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("[MOCK WEBHOOK] Error reading body: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[MOCK WEBHOOK RECEIVED PAYLOAD]\n%s\n", string(body))
		c.JSON(http.StatusOK, gin.H{"status": "delivered"})
	})

	log.Printf("Starting Server on port %s\n", appPort)
	if err := r.Run(":" + appPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func startPurgeWorker(ctx context.Context, dbClient *database.Client, lokiClient *loki.Client, interval, retentionPeriod time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting background purge worker. Run interval: %v, Retention period: %v\n", interval, retentionPeriod)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping background purge worker.")
			return
		case <-ticker.C:
			log.Println("Purge worker ticker fired. Executing database and Loki purge...")
			cutoffTime := time.Now().Add(-retentionPeriod)

			// 1. Purge PostgreSQL entries
			rowsDeleted, err := dbClient.PurgeRecords(ctx, cutoffTime)
			if err != nil {
				log.Printf("Purge worker error: failed to purge PostgreSQL records: %v\n", err)
			} else {
				log.Printf("Purge worker success: deleted %d PostgreSQL execution records older than %v\n", rowsDeleted, cutoffTime)
			}

			// 2. Purge Loki logs
			epochStart := time.Now().AddDate(-5, 0, 0)
			err = lokiClient.DeleteLogs(ctx, epochStart, cutoffTime)
			if err != nil {
				log.Printf("Purge worker error: failed to delete Loki logs: %v\n", err)
			} else {
				log.Printf("Purge worker success: requested Loki logs deletion older than %v\n", cutoffTime)
			}
		}
	}
}

