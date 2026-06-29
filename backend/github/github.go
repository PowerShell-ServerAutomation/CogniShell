package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v62/github"
)

// Connector defines operations to interact with GitHub
type Connector interface {
	FetchScript(ctx context.Context, owner, repo, path, ref string) (string, error)
	CreateIssue(ctx context.Context, owner, repo, title, body string, assignees []string) (*github.Issue, error)
	FetchCODEOWNERS(ctx context.Context, owner, repo string) (string, error)
}

// Client implements Connector using go-github SDK
type Client struct {
	ghClient *github.Client
}

// NewClient initializes a new GitHub client. If token is empty, it uses an unauthenticated client.
func NewClient(token string) *Client {
	var httpClient *http.Client
	ghClient := github.NewClient(httpClient)
	if token != "" {
		ghClient = ghClient.WithAuthToken(token)
	}
	return &Client{
		ghClient: ghClient,
	}
}

// FetchScript downloads a script from GitHub into memory
func (c *Client) FetchScript(ctx context.Context, owner, repo, path, ref string) (string, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: ref,
	}
	
	// DownloadContents retrieves the raw file stream directly
	rc, _, err := c.ghClient.Repositories.DownloadContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("failed to download content from github: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read github content stream: %w", err)
	}

	return string(content), nil
}

// CreateIssue creates a new issue in GitHub and assigns it to specified assignees
func (c *Client) CreateIssue(ctx context.Context, owner, repo, title, body string, assignees []string) (*github.Issue, error) {
	req := &github.IssueRequest{
		Title: &title,
		Body:  &body,
	}

	if len(assignees) > 0 {
		var cleaned []string
		for _, a := range assignees {
			cName := strings.TrimPrefix(a, "@")
			if cName != "" {
				cleaned = append(cleaned, cName)
			}
		}
		if len(cleaned) > 0 {
			req.Assignees = &cleaned
		}
	}

	issue, _, err := c.ghClient.Issues.Create(ctx, owner, repo, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	return issue, nil
}

// FetchCODEOWNERS searches for CODEOWNERS file in standard paths and downloads it
func (c *Client) FetchCODEOWNERS(ctx context.Context, owner, repo string) (string, error) {
	paths := []string{"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS"}
	var lastErr error
	for _, p := range paths {
		opts := &github.RepositoryContentGetOptions{
			Ref: "main",
		}
		rc, _, err := c.ghClient.Repositories.DownloadContents(ctx, owner, repo, p, opts)
		if err == nil {
			defer rc.Close()
			content, err := io.ReadAll(rc)
			if err == nil {
				return string(content), nil
			}
			lastErr = err
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("failed to retrieve CODEOWNERS: %w", lastErr)
}

// ParseCODEOWNERS extracts the owners for a matching path pattern
func ParseCODEOWNERS(content, scriptPath string) []string {
	lines := strings.Split(content, "\n")
	var matchingOwners []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) < 2 {
			continue
		}

		pattern := parts[0]
		// Simple pattern matching: matches wildcard '*' or if scriptPath contains pattern prefix (without leading slash)
		cleanPattern := strings.TrimPrefix(pattern, "/")
		if pattern == "*" || strings.Contains(scriptPath, cleanPattern) {
			matchingOwners = parts[1:]
		}
	}

	return matchingOwners
}
