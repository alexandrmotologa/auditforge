package forwarder

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

func TestForwarder_HMACSignatureAndDelivery(t *testing.T) {
	var receivedBody []byte
	var receivedSignature string
	deliveredChan := make(chan struct{}, 1)

	secret := "secret-compliance-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-AuditForge-Signature")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		deliveredChan <- struct{}{}
	}))
	defer server.Close()

	fwd := NewForwarder()
	fwd.RegisterSubscription(&Subscription{
		ID:            "sub_test",
		Name:          "SIEM Test Receiver",
		URL:           server.URL,
		Secret:        secret,
		TenantID:      "org_acme",
		FilterActions: []string{"permission.grant"},
		Enabled:       true,
	})

	events := []*ledger.AuditEvent{
		{
			ID:        "evt_fwd_1",
			Timestamp: time.Now().UTC(),
			TenantID:  "org_acme",
			Action:    "permission.grant",
			Actor:     ledger.Actor{ID: "usr_alice"},
			Target:    ledger.Target{ID: "vault_1"},
		},
		{
			ID:        "evt_fwd_2",
			Timestamp: time.Now().UTC(),
			TenantID:  "org_other", // should be filtered out
			Action:    "permission.grant",
			Actor:     ledger.Actor{ID: "usr_bob"},
			Target:    ledger.Target{ID: "vault_2"},
		},
	}

	block, _, _ := ledger.NewBlock(1, make([]byte, 32), events, ledger.AlgorithmBlake3)
	_ = block.Seal(nil)

	fwd.ForwardBlock(block)

	select {
	case <-deliveredChan:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatalf("webhook not delivered in time")
	}

	// Verify HMAC-SHA256 signature
	if !strings.HasPrefix(receivedSignature, "sha256=") {
		t.Fatalf("missing or invalid signature prefix: %s", receivedSignature)
	}

	expectedSigHex := strings.TrimPrefix(receivedSignature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(receivedBody)
	calculatedSigHex := hex.EncodeToString(mac.Sum(nil))

	if expectedSigHex != calculatedSigHex {
		t.Fatalf("hmac signature mismatch: expected %s, got %s", expectedSigHex, calculatedSigHex)
	}
}
