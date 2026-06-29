package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type LokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

type LokiPushRequest struct {
	Streams []LokiStream `json:"streams"`
}

func (c *Client) PushLogs(ctx context.Context, scriptName, executionID, environment, stdout, stderr string) error {
	var streams []LokiStream
	nowStr := fmt.Sprintf("%d", time.Now().UnixNano())

	if strings.TrimSpace(stdout) != "" {
		lines := strings.Split(stdout, "\n")
		var values [][]string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			values = append(values, []string{nowStr, line})
		}
		if len(values) > 0 {
			streams = append(streams, LokiStream{
				Stream: map[string]string{
					"script_name":  scriptName,
					"execution_id": executionID,
					"environment":  environment,
					"stream":       "stdout",
				},
				Values: values,
			})
		}
	}

	if strings.TrimSpace(stderr) != "" {
		lines := strings.Split(stderr, "\n")
		var values [][]string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			values = append(values, []string{nowStr, line})
		}
		if len(values) > 0 {
			streams = append(streams, LokiStream{
				Stream: map[string]string{
					"script_name":  scriptName,
					"execution_id": executionID,
					"environment":  environment,
					"stream":       "stderr",
				},
				Values: values,
			})
		}
	}

	if len(streams) == 0 {
		return nil
	}

	reqBody := LokiPushRequest{Streams: streams}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal Loki request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/loki/api/v1/push", c.url)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("loki returned unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) DeleteLogs(ctx context.Context, start, end time.Time) error {
	form := url.Values{}
	form.Set("query", `{environment=~".*"}`)
	form.Set("start", fmt.Sprintf("%d", start.Unix()))
	form.Set("end", fmt.Sprintf("%d", end.Unix()))

	reqURL := fmt.Sprintf("%s/loki/api/v1/delete?%s", c.url, form.Encode())
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create Loki delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger Loki delete logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("loki delete logs returned unexpected status: %d, body: %s", resp.StatusCode, buf.String())
	}
	return nil
}
