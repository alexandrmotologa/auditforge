package auth

import (
	"path/filepath"
	"testing"
)

func TestKeyStore_RBACAndPermissions(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "keys.json")

	ks, err := NewKeyStore(storePath, []string{"master-admin-key"})
	if err != nil {
		t.Fatalf("failed initializing keystore: %v", err)
	}

	// 1. Verify default admin key
	adminKey, ok := ks.Authenticate("master-admin-key")
	if !ok || adminKey.Role != RoleAdmin {
		t.Fatalf("expected admin key to authenticate")
	}

	// 2. Create ingest_only key for org_acme
	ingestKey, err := ks.CreateKey("Ingest Service", "org_acme", RoleIngestOnly)
	if err != nil {
		t.Fatalf("failed creating ingest key: %v", err)
	}

	// 3. Create auditor_read key for org_acme
	auditorKey, err := ks.CreateKey("Auditor Read", "org_acme", RoleAuditorRead)
	if err != nil {
		t.Fatalf("failed creating auditor key: %v", err)
	}

	// 4. Verify Role Isolation
	// Ingest key can ingest for org_acme
	if !HasPermission(ingestKey, RoleIngestOnly, "org_acme") {
		t.Errorf("ingest key should have ingest permission for org_acme")
	}
	// Ingest key CANNOT read
	if HasPermission(ingestKey, RoleAuditorRead, "org_acme") {
		t.Errorf("ingest key should NOT have auditor_read permission")
	}
	// Ingest key CANNOT ingest for org_other
	if HasPermission(ingestKey, RoleIngestOnly, "org_other") {
		t.Errorf("ingest key for org_acme should NOT have permission for org_other")
	}

	// Auditor key CAN read org_acme
	if !HasPermission(auditorKey, RoleAuditorRead, "org_acme") {
		t.Errorf("auditor key should have auditor_read permission for org_acme")
	}
	// Auditor key CANNOT ingest
	if HasPermission(auditorKey, RoleIngestOnly, "org_acme") {
		t.Errorf("auditor key should NOT have ingest_only permission")
	}

	// Admin key CAN do both on any tenant
	if !HasPermission(adminKey, RoleIngestOnly, "org_acme") || !HasPermission(adminKey, RoleAuditorRead, "org_other") {
		t.Errorf("admin key should have universal access")
	}

	// 5. Test Revocation
	if err := ks.RevokeKey(ingestKey.ID); err != nil {
		t.Fatalf("revocation failed: %v", err)
	}
	if _, ok := ks.Authenticate(ingestKey.Key); ok {
		t.Fatalf("revoked key should not authenticate")
	}
}
