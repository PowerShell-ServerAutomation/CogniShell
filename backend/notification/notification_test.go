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
	)

	if err != nil {
		t.Fatalf("SendNotification() returned error: %v", err)
	}

	if receivedPayload["type"] != "message" {
		t.Errorf("Expected payload type to be 'message', got %v", receivedPayload["type"])
	}

	attachments, ok := receivedPayload["attachments"].([]interface{})
	if !ok || len(attachments) == 0 {
		t.Fatal("Expected attachments slice to be present")
	}

	attachment := attachments[0].(map[string]interface{})
	if attachment["contentType"] != "application/vnd.microsoft.card.adaptive" {
		t.Errorf("Expected attachment contentType to be 'application/vnd.microsoft.card.adaptive', got %v", attachment["contentType"])
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
	)

	if err != nil {
		t.Fatalf("SendNotification() returned error: %v", err)
	}

	blocks, ok := receivedPayload["blocks"].([]interface{})
	if !ok || len(blocks) == 0 {
		t.Fatal("Expected blocks slice to be present")
	}
}
