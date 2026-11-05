package auditforge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Actor represents an identity initiating an audit action
type Actor struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Email    string         `json:"email,omitempty"`
	Role     string         `json:"role,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Target represents the resource being operated upon
type Target struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Event represents an audit event payload to emit
type Event struct {
	ID        string         `json:"id,omitempty"`
	Timestamp time.Time      `json:"timestamp,omitempty"`
	TenantID  string         `json:"tenant_id,omitempty"`
	Actor     Actor          `json:"actor"`
	Action    string         `json:"action"`
	Target    Target         `json:"target"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Config configures the Go client SDK
type Config struct {
	Endpoint      string
	APIKey        string
	TenantID      string
	BatchSize     int
	FlushInterval time.Duration
}

// Client provides high-throughput, non-blocking audit logging
type Client struct {
	mu         sync.Mutex
	cfg        Config
	httpClient *http.Client
	queue      []*Event
	ticker     *time.Ticker
	quit       chan struct{}
}

// NewClient initializes the AuditForge client with background batching
func NewClient(cfg Config) *Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://localhost:8080"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 200 * time.Millisecond
	}

	c := &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		queue:      make([]*Event, 0, cfg.BatchSize),
		ticker:     time.NewTicker(cfg.FlushInterval),
		quit:       make(chan struct{}),
	}

	go c.worker()
	return c
}

// Emit queues an event for non-blocking batched ingestion
func (c *Client) Emit(evt *Event) {
	if evt.TenantID == "" {
		evt.TenantID = c.cfg.TenantID
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	c.mu.Lock()
	c.queue = append(c.queue, evt)
	shouldFlush := len(c.queue) >= c.cfg.BatchSize
	c.mu.Unlock()

	if shouldFlush {
		go c.Flush()
	}
}

// EmitSync sends an event synchronously and waits for block sealing
func (c *Client) EmitSync(ctx context.Context, evt *Event) error {
	if evt.TenantID == "" {
		evt.TenantID = c.cfg.TenantID
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	body, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/events?sync=true", c.cfg.Endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ingest sync failed with status %d", resp.StatusCode)
	}
	return nil
}

// Flush sends all buffered events immediately
func (c *Client) Flush() error {
	c.mu.Lock()
	if len(c.queue) == 0 {
		c.mu.Unlock()
		return nil
	}

	batch := c.queue
	c.queue = make([]*Event, 0, c.cfg.BatchSize)
	c.mu.Unlock()

	payload := map[string]any{
		"tenant_id": c.cfg.TenantID,
		"events":    batch,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/events/batch", c.cfg.Endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("batch ingest failed with status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) worker() {
	for {
		select {
		case <-c.quit:
			return
		case <-c.ticker.C:
			_ = c.Flush()
		}
	}
}

// Close gracefully flushes remaining events and terminates background worker
func (c *Client) Close() error {
	close(c.quit)
	c.ticker.Stop()
	return c.Flush()
}
