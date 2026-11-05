package forwarder

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// Subscription defines a target webhook endpoint for real-time audit forwarding
type Subscription struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Secret         string    `json:"secret"` // Shared secret for HMAC-SHA256 signature
	TenantID       string    `json:"tenant_id"` // specific tenant or "*"
	FilterActions  []string  `json:"filter_actions,omitempty"` // empty for all
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	DeliveredCount uint64    `json:"delivered_count"`
	FailedCount    uint64    `json:"failed_count"`
	LastStatus     int       `json:"last_status,omitempty"`
}

// Forwarder manages SIEM and webhook streaming
type Forwarder struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
	httpClient    *http.Client
}

// NewForwarder initializes the webhook streaming forwarder
func NewForwarder() *Forwarder {
	return &Forwarder{
		subscriptions: make(map[string]*Subscription),
		httpClient:    &http.Client{Timeout: 8 * time.Second},
	}
}

// RegisterSubscription adds or updates a webhook destination
func (f *Forwarder) RegisterSubscription(sub *Subscription) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now().UTC()
	}
	f.subscriptions[sub.ID] = sub
}

// ListSubscriptions returns active destinations
func (f *Forwarder) ListSubscriptions() []*Subscription {
	f.mu.RLock()
	defer f.mu.RUnlock()

	list := make([]*Subscription, 0, len(f.subscriptions))
	for _, sub := range f.subscriptions {
		cp := *sub
		list = append(list, &cp)
	}
	return list
}

// ForwardBlock evaluates all subscriptions and dispatches matching events
func (f *Forwarder) ForwardBlock(block *ledger.Block) {
	f.mu.RLock()
	subs := make([]*Subscription, 0, len(f.subscriptions))
	for _, sub := range f.subscriptions {
		if sub.Enabled {
			subs = append(subs, sub)
		}
	}
	f.mu.RUnlock()

	for _, sub := range subs {
		matchedEvents := filterEvents(block.Events, sub)
		if len(matchedEvents) == 0 {
			continue
		}

		go f.dispatchBatch(sub, block.Header, matchedEvents)
	}
}

func filterEvents(events []*ledger.AuditEvent, sub *Subscription) []*ledger.AuditEvent {
	var matched []*ledger.AuditEvent
	for _, e := range events {
		if sub.TenantID != "*" && sub.TenantID != "" && e.TenantID != sub.TenantID {
			continue
		}

		if len(sub.FilterActions) > 0 {
			actionMatch := false
			for _, act := range sub.FilterActions {
				if act == e.Action || act == "*" {
					actionMatch = true
					break
				}
			}
			if !actionMatch {
				continue
			}
		}

		matched = append(matched, e)
	}
	return matched
}

func (f *Forwarder) dispatchBatch(sub *Subscription, header ledger.BlockHeader, events []*ledger.AuditEvent) {
	payload := map[string]any{
		"block_index": header.Index,
		"block_hash":  header.BlockHashHex,
		"merkle_root": header.RootHex,
		"timestamp":   header.Timestamp.UTC().Format(time.RFC3339Nano),
		"event_count": len(events),
		"events":      events,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", sub.URL, bytes.NewReader(body))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AuditForge-Block", fmt.Sprintf("%d", header.Index))

	// HMAC-SHA256 signature
	if sub.Secret != "" {
		mac := hmac.New(sha256.New, []byte(sub.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-AuditForge-Signature", fmt.Sprintf("sha256=%s", sig))
	}

	resp, err := f.httpClient.Do(req)
	f.mu.Lock()
	defer f.mu.Unlock()

	if err != nil || resp.StatusCode >= 400 {
		sub.FailedCount++
		if resp != nil {
			sub.LastStatus = resp.StatusCode
			_ = resp.Body.Close()
		}
	} else {
		sub.DeliveredCount += uint64(len(events))
		sub.LastStatus = resp.StatusCode
		_ = resp.Body.Close()
	}
}
