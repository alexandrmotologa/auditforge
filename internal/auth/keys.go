package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	RoleIngestOnly  = "ingest_only"
	RoleAuditorRead = "auditor_read"
	RoleAdmin       = "admin"
)

// APIKey encapsulates authentication credentials, tenant isolation, and RBAC role
type APIKey struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	TenantID   string     `json:"tenant_id"` // specific tenant or "*" for all tenants
	Role       string     `json:"role"`      // ingest_only, auditor_read, admin
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// KeyStore manages persistent API keys with role-based access control
type KeyStore struct {
	mu       sync.RWMutex
	filePath string
	keys     map[string]*APIKey // key string -> APIKey
	byId     map[string]*APIKey // id -> APIKey
}

// NewKeyStore loads or initializes the API key store
func NewKeyStore(filePath string, defaultKeys []string) (*KeyStore, error) {
	ks := &KeyStore{
		filePath: filePath,
		keys:     make(map[string]*APIKey),
		byId:     make(map[string]*APIKey),
	}

	if filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, fmt.Errorf("failed creating auth directory: %w", err)
		}
		if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
			var loaded []*APIKey
			if err := json.Unmarshal(data, &loaded); err == nil {
				for _, k := range loaded {
					ks.keys[k.Key] = k
					ks.byId[k.ID] = k
				}
			}
		}
	}

	// Ensure default keys exist if store is empty
	if len(ks.keys) == 0 {
		if len(defaultKeys) > 0 {
			for i, rawKey := range defaultKeys {
				k := &APIKey{
					ID:        fmt.Sprintf("key_default_%d", i+1),
					Key:       rawKey,
					Name:      fmt.Sprintf("Default Key %d", i+1),
					TenantID:  "*",
					Role:      RoleAdmin,
					CreatedAt: time.Now().UTC(),
				}
				ks.keys[rawKey] = k
				ks.byId[k.ID] = k
			}
		} else {
			// Create a default admin dev key
			devKey := "default-dev-key"
			k := &APIKey{
				ID:        "key_default_admin",
				Key:       devKey,
				Name:      "Default Development Admin Key",
				TenantID:  "*",
				Role:      RoleAdmin,
				CreatedAt: time.Now().UTC(),
			}
			ks.keys[devKey] = k
			ks.byId[k.ID] = k
		}
		_ = ks.saveLocked()
	}

	return ks, nil
}

// Authenticate verifies the token and updates the last used timestamp
func (ks *KeyStore) Authenticate(token string) (*APIKey, bool) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	k, exists := ks.keys[token]
	if !exists {
		return nil, false
	}

	now := time.Now().UTC()
	k.LastUsedAt = &now
	return k, true
}

// HasPermission checks if the API key satisfies the required role and tenant constraint
func HasPermission(k *APIKey, requiredRole, targetTenant string) bool {
	if k == nil {
		return false
	}

	// Super-admin has universal access
	if k.Role == RoleAdmin {
		return true
	}

	// Role match
	if requiredRole != "" && k.Role != requiredRole {
		return false
	}

	// Tenant match (unless key has universal tenant access "*")
	if targetTenant != "" && k.TenantID != "*" && k.TenantID != targetTenant {
		return false
	}

	return true
}

// CreateKey provisions a new API key with specific role and tenant scope
func (ks *KeyStore) CreateKey(name, tenantID, role string) (*APIKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if role != RoleIngestOnly && role != RoleAuditorRead && role != RoleAdmin {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	if tenantID == "" {
		tenantID = "*"
	}

	randBytes := make([]byte, 18)
	_, _ = rand.Read(randBytes)
	keyStr := fmt.Sprintf("af_live_%s", hex.EncodeToString(randBytes))

	idBytes := make([]byte, 6)
	_, _ = rand.Read(idBytes)
	id := fmt.Sprintf("key_%s", hex.EncodeToString(idBytes))

	apiKey := &APIKey{
		ID:        id,
		Key:       keyStr,
		Name:      name,
		TenantID:  tenantID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}

	ks.keys[keyStr] = apiKey
	ks.byId[id] = apiKey

	_ = ks.saveLocked()
	return apiKey, nil
}

// ListKeys returns all active registered keys
func (ks *KeyStore) ListKeys() []*APIKey {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	list := make([]*APIKey, 0, len(ks.byId))
	for _, k := range ks.byId {
		// Return copy
		copied := *k
		list = append(list, &copied)
	}
	return list
}

// RevokeKey disables an API key by ID
func (ks *KeyStore) RevokeKey(id string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	k, exists := ks.byId[id]
	if !exists {
		return fmt.Errorf("key with id %s not found", id)
	}

	delete(ks.keys, k.Key)
	delete(ks.byId, id)

	return ks.saveLocked()
}

func (ks *KeyStore) saveLocked() error {
	if ks.filePath == "" {
		return nil
	}

	list := make([]*APIKey, 0, len(ks.byId))
	for _, k := range ks.byId {
		list = append(list, k)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ks.filePath, data, 0600)
}
