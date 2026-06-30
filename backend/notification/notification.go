package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendNotification routes the alert to the selected provider (teams | slack)
func (c *Client) SendNotification(ctx context.Context, provider, webhookURL, scriptPath, environment string, assignees []string, issueURL string) error {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" || provider == "none" {
		return nil
	}

	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty for provider: %s", provider)
	}

	switch provider {
	case "teams":
		return c.sendTeamsAdaptiveCard(ctx, webhookURL, scriptPath, environment, assignees, issueURL)
	case "slack":
		return c.sendSlackBlocks(ctx, webhookURL, scriptPath, environment, assignees, issueURL)
	default:
		return fmt.Errorf("unsupported notification provider: %s", provider)
	}
}

func (c *Client) sendTeamsAdaptiveCard(ctx context.Context, webhookURL, scriptPath, environment string, assignees []string, issueURL string) error {
	assigneesStr := strings.Join(assignees, ", ")
	if assigneesStr == "" {
		assigneesStr = "unassigned"
	}

	payload := map[string]interface{}{
		"type":        "AdaptiveCard",
		"version":     "1.4",
		"$schema":     "http://adaptivecards.io/schemas/adaptive-card.json",
		"body": []interface{}{
			map[string]interface{}{
				"type":   "TextBlock",
				"size":   "medium",
				"weight": "bolder",
				"text":   "🚨 CogniShell Alert: Script Execution Failure",
				"color":  "Attention",
			},
			map[string]interface{}{
				"type":   "TextBlock",
				"text":   "An automated script execution failed with a non-zero exit code. An issue has been created and assigned.",
				"wrap":   true,
				"spacing": "small",
			},
			map[string]interface{}{
				"type": "FactSet",
				"facts": []interface{}{
					map[string]interface{}{"title": "Script:", "value": scriptPath},
					map[string]interface{}{"title": "Environment:", "value": environment},
					map[string]interface{}{"title": "Assignees:", "value": assigneesStr},
				},
				"spacing": "medium",
			},
		},
		"actions": []interface{}{
			map[string]interface{}{
				"type":  "Action.OpenUrl",
				"title": "View GitHub Issue",
				"url":   issueURL,
			},
		},
	}

	return c.postJSON(ctx, webhookURL, payload)
}

func (c *Client) sendSlackBlocks(ctx context.Context, webhookURL, scriptPath, environment string, assignees []string, issueURL string) error {
	assigneesStr := strings.Join(assignees, ", ")
	if assigneesStr == "" {
		assigneesStr = "unassigned"
	}

	payload := map[string]interface{}{
		"blocks": []interface{}{
			map[string]interface{}{
				"type": "header",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": "🚨 CogniShell Alert: Script Execution Failure",
				},
			},
			map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": "An automated script execution failed with a non-zero exit code. An issue has been created and assigned.",
				},
			},
			map[string]interface{}{
				"type": "section",
				"fields": []interface{}{
					map[string]interface{}{"type": "mrkdwn", "text": fmt.Sprintf("*Script:*\n`%s`", scriptPath)},
					map[string]interface{}{"type": "mrkdwn", "text": fmt.Sprintf("*Environment:*\n`%s`", environment)},
					map[string]interface{}{"type": "mrkdwn", "text": fmt.Sprintf("*Assignees:*\n%s", assigneesStr)},
				},
			},
			map[string]interface{}{
				"type": "actions",
				"elements": []interface{}{
					map[string]interface{}{
						"type": "button",
						"text": map[string]interface{}{
							"type": "plain_text",
							"text": "View GitHub Issue",
						},
						"url":   issueURL,
						"style": "danger",
					},
				},
			},
		},
	}

	return c.postJSON(ctx, webhookURL, payload)
}

func (c *Client) postJSON(ctx context.Context, url string, payload interface{}) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// MS Teams webhooks can return 200 OK or 201 Created or even simple success strings
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook endpoint returned unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
