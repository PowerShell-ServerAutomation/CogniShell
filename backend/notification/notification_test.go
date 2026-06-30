package notification

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendNotificationTeams(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal("Failed to read request body")
		}

		err = json.Unmarshal(body, &receivedPayload)
		if err != nil {
			t.Fatal("Failed to unmarshal body")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	err := client.SendNotification(
		context.Background(),
		"teams",
		server.URL,
		"scripts/hello-world/hello-world.ps1",
		"dev",
		[]string{"@vsrivastava", "@johndoe"},
		"https://github.com/issue/1",
		false,
	)

	if err != nil {
		t.Fatalf("SendNotification() returned error: %v", err)
	}

	if receivedPayload["type"] != "AdaptiveCard" {
		t.Errorf("Expected payload type to be 'AdaptiveCard', got %v", receivedPayload["type"])
	}

	if receivedPayload["version"] != "1.4" {
		t.Errorf("Expected version to be '1.4', got %v", receivedPayload["version"])
	}
}

func TestSendNotificationSlack(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal("Failed to read request body")
		}

		err = json.Unmarshal(body, &receivedPayload)
		if err != nil {
			t.Fatal("Failed to unmarshal body")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	err := client.SendNotification(
		context.Background(),
		"slack",
		server.URL,
		"scripts/hello-world/hello-world.ps1",
		"dev",
		[]string{"@vsrivastava"},
		"https://github.com/issue/1",
		false,
	)

	if err != nil {
		t.Fatalf("SendNotification() returned error: %v", err)
	}

	blocks, ok := receivedPayload["blocks"].([]interface{})
	if !ok || len(blocks) == 0 {
		t.Fatal("Expected blocks slice to be present")
	}
}
