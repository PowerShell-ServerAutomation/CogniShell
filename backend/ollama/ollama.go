package ollama

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
	url        string
	httpClient *http.Client
}

func NewClient(url string) *Client {
	return &Client{
		url:        strings.TrimSuffix(url, "/"),
		httpClient: &http.Client{Timeout: 150 * time.Second}, // Longer timeout since LLM inference on a single CPU core can take some time
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
}

func (c *Client) AnalyzeError(ctx context.Context, scriptPath, scriptCode, stderr string) (string, error) {
	prompt := fmt.Sprintf(
		"You are an expert DevOps engineer and PowerShell debugger.\n"+
			"Analyze the following PowerShell script failure and provide a root cause analysis and specific remediation steps.\n\n"+
			"Script Path: %s\n\n"+
			"--- SCRIPT CODE ---\n%s\n\n"+
			"--- STDERR ERROR TRACE ---\n%s\n\n"+
			"Please respond in clear Markdown format with two main sections: '## Root Cause Analysis' and '## Suggested Remediation'.",
		scriptPath, scriptCode, stderr,
	)

	reqBody := GenerateRequest{
		Model:  "qwen2.5-coder:0.5b",
		Prompt: prompt,
		Stream: false,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/api/generate", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned unexpected status code: %d", resp.StatusCode)
	}

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return genResp.Response, nil
}
