package auditforge_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	auditforge "github.com/alexandrmotologa/auditforge/sdk/go"
)

func TestGoSDK_EmitAndFlush(t *testing.T) {
	var batchReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events/batch" {
			var payload map[string]any
			_ = json.NewDecoder(r.Body).Decode(&payload)
			events := payload["events"].([]any)
			if len(events) == 2 {
				batchReceived = true
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := auditforge.NewClient(auditforge.Config{
		Endpoint:      server.URL,
		TenantID:      "org_sdk_test",
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
	})
	defer client.Close()

	client.Emit(&auditforge.Event{
		Action: "user.signup",
		Actor:  auditforge.Actor{ID: "usr_1", Name: "Tester"},
		Target: auditforge.Target{ID: "account_1", Type: "account"},
	})
	client.Emit(&auditforge.Event{
		Action: "user.login",
		Actor:  auditforge.Actor{ID: "usr_1", Name: "Tester"},
		Target: auditforge.Target{ID: "session_1", Type: "session"},
	})

	err := client.Flush()
	if err != nil {
		t.Fatalf("client flush failed: %v", err)
	}

	if !batchReceived {
		t.Fatalf("expected batch of 2 events to be received by test server")
	}
}

func TestGoSDK_EmitSync(t *testing.T) {
	var syncReceived bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events" && r.URL.Query().Get("sync") == "true" {
			syncReceived = true
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := auditforge.NewClient(auditforge.Config{
		Endpoint: server.URL,
		TenantID: "org_sync_test",
	})
	defer client.Close()

	err := client.EmitSync(context.Background(), &auditforge.Event{
		Action: "role.grant",
		Actor:  auditforge.Actor{ID: "usr_admin"},
		Target: auditforge.Target{ID: "perm_write"},
	})
	if err != nil {
		t.Fatalf("emit sync failed: %v", err)
	}

	if !syncReceived {
		t.Fatalf("expected sync request to be received")
	}
}
