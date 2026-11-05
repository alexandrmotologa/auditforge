package ledger

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Actor represents the user, service account, or system component initiating the action
type Actor struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Email    string         `json:"email,omitempty"`
	Role     string         `json:"role,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Target represents the resource or entity being affected by the action
type Target struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AuditEvent represents a single immutable compliance audit record
type AuditEvent struct {
	ID        string         `json:"id"`
	Timestamp time.Time      `json:"timestamp"`
	TenantID  string         `json:"tenant_id"`
	Actor     Actor          `json:"actor"`
	Action    string         `json:"action"`
	Target    Target         `json:"target"`
	Metadata  map[string]any `json:"metadata,omitempty"`

	// Cryptographic linkage fields
	LeafHash   string `json:"leaf_hash,omitempty"`
	BlockIndex uint64 `json:"block_index,omitempty"`
	LeafIndex  int    `json:"leaf_index,omitempty"`
}

// CanonicalPayload returns the deterministic structure used for cryptographic hashing.
// Excludes BlockIndex and LeafIndex so the event hash is invariant regardless of when or where it is sealed.
type CanonicalPayload struct {
	Action    string         `json:"action"`
	Actor     Actor          `json:"actor"`
	ID        string         `json:"id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Target    Target         `json:"target"`
	TenantID  string         `json:"tenant_id"`
	Timestamp string         `json:"timestamp"`
}

// NewEventID generates a collision-resistant event identifier with an evt_ prefix
func NewEventID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("evt_%s", hex.EncodeToString(b))
}

// CanonicalBytes returns the RFC 8785 canonical JSON bytes of the event
func (e *AuditEvent) CanonicalBytes() ([]byte, error) {
	// Format timestamp strictly to UTC RFC 3339 nano for deterministic representation
	ts := e.Timestamp.UTC().Format(time.RFC3339Nano)
	payload := CanonicalPayload{
		ID:        e.ID,
		Timestamp: ts,
		TenantID:  e.TenantID,
		Actor:     e.Actor,
		Action:    e.Action,
		Target:    e.Target,
		Metadata:  e.Metadata,
	}

	return CanonicalizeJSON(payload)
}

// ComputeLeafHash computes the cryptographic leaf hash for this event
func (e *AuditEvent) ComputeLeafHash(algorithm string) ([]byte, error) {
	canonicalBytes, err := e.CanonicalBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize event %s: %w", e.ID, err)
	}

	hashBytes := HashLeaf(algorithm, canonicalBytes)
	e.LeafHash = FormatHashHex(hashBytes)
	return hashBytes, nil
}
