package ledger

import (
	"testing"
	"time"
)

func createTestEvent(id string, action string) *AuditEvent {
	return &AuditEvent{
		ID:        id,
		Timestamp: time.Now().UTC(),
		TenantID:  "org_test",
		Actor: Actor{
			ID:    "usr_1",
			Name:  "Test User",
			Email: "test@example.com",
			Role:  "admin",
		},
		Action: action,
		Target: Target{
			ID:   "res_1",
			Type: "document",
			Name: "Confidential Report",
		},
		Metadata: map[string]any{
			"ip": "127.0.0.1",
		},
	}
}

func TestBlock_SealingAndIntegrity(t *testing.T) {
	signer, err := NewEd25519Signer()
	if err != nil {
		t.Fatalf("failed creating signer: %v", err)
	}

	events := []*AuditEvent{
		createTestEvent("evt_1", "user.login"),
		createTestEvent("evt_2", "doc.read"),
		createTestEvent("evt_3", "doc.export"),
	}

	block, _, err := NewBlock(1, make([]byte, 32), events, AlgorithmBlake3)
	if err != nil {
		t.Fatalf("failed creating block: %v", err)
	}

	if err := block.Seal(signer); err != nil {
		t.Fatalf("failed sealing block: %v", err)
	}

	// Verify intact block
	valid, err := block.VerifyIntegrity()
	if !valid || err != nil {
		t.Fatalf("valid block failed integrity check: %v", err)
	}

	// Tamper: modify event action in-place
	events[1].Action = "doc.delete"
	valid, err = block.VerifyIntegrity()
	if valid {
		t.Fatalf("tampered event action passed integrity verification!")
	}

	// Restore event, tamper with block header root
	events[1].Action = "doc.read"
	_, _ = events[1].ComputeLeafHash(AlgorithmBlake3)
	block.Header.MerkleRoot[0] ^= 0xAA

	valid, err = block.VerifyIntegrity()
	if valid {
		t.Fatalf("tampered merkle root passed integrity verification!")
	}
}
