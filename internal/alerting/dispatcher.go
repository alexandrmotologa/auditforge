package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AlertPayload defines the structured notification sent when an integrity violation is detected
type AlertPayload struct {
	Level              string    `json:"level"` // CRITICAL, WARNING, INFO
	Title              string    `json:"title"`
	Message            string    `json:"message"`
	TamperedBlockIndex *uint64   `json:"tampered_block_index,omitempty"`
	Reason             string    `json:"reason,omitempty"`
	DetectedAt         time.Time `json:"detected_at"`
	Hostname           string    `json:"hostname"`
}

// Destination represents an external webhook endpoint (e.g. Slack, PagerDuty, Discord, Custom)
type Destination struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Enabled bool              `json:"enabled"`
	Headers map[string]string `json:"headers,omitempty"`
}

// Dispatcher coordinates multi-channel tamper alert notifications
type Dispatcher struct {
	mu           sync.RWMutex
	destinations []*Destination
	httpClient   *http.Client
}

// NewDispatcher initializes an alert dispatcher
func NewDispatcher(destinations []*Destination) *Dispatcher {
	return &Dispatcher{
		destinations: destinations,
		httpClient:   &http.Client{Timeout: 5 * time.Second},
	}
}

// AddDestination registers a new alert target
func (d *Dispatcher) AddDestination(dest *Destination) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.destinations = append(d.destinations, dest)
}

// NotifyTamper sends a high-priority alert across all registered channels
func (d *Dispatcher) NotifyTamper(blockIndex uint64, reason string) {
	payload := AlertPayload{
		Level:              "CRITICAL",
		Title:              "AuditForge Cryptographic Integrity Violation",
		Message:            fmt.Sprintf("Immediate action required: Merkle DAG chain severed at block #%d. Retroactive alteration detected.", blockIndex),
		TamperedBlockIndex: &blockIndex,
		Reason:             reason,
		DetectedAt:         time.Now().UTC(),
		Hostname:           "auditforge-ledger-node",
	}

	d.mu.RLock()
	destList := append([]*Destination(nil), d.destinations...)
	d.mu.RUnlock()

	for _, dest := range destList {
		if !dest.Enabled || dest.URL == "" {
			continue
		}
		go d.send(dest, payload)
	}
}

func (d *Dispatcher) send(dest *Destination, payload AlertPayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", dest.URL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range dest.Headers {
		req.Header.Set(k, v)
	}

	resp, err := d.httpClient.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
