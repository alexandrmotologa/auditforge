package ledger

import (
	"testing"
)

func TestVerifyLedgerChain_DetectsBrokenChain(t *testing.T) {
	signer, _ := NewEd25519Signer()
	var blocks []*Block
	var prevHash []byte = make([]byte, 32)

	for i := uint64(0); i < 4; i++ {
		evts := []*AuditEvent{createTestEvent(NewEventID(), "user.login")}
		b, _, err := NewBlock(i, prevHash, evts, AlgorithmBlake3)
		if err != nil {
			t.Fatalf("failed creating block: %v", err)
		}
		if err := b.Seal(signer); err != nil {
			t.Fatalf("failed sealing block: %v", err)
		}
		blocks = append(blocks, b)
		prevHash = b.Header.BlockHash
	}

	// Initial chain is valid
	res := VerifyLedgerChain(blocks)
	if !res.Valid {
		t.Fatalf("initial chain should be valid: %s", res.Reason)
	}

	// Tamper: modify event inside Block #2
	blocks[2].Events[0].Action = "tampered.action"
	_, _ = blocks[2].Events[0].ComputeLeafHash(AlgorithmBlake3)

	resTampered := VerifyLedgerChain(blocks)
	if resTampered.Valid {
		t.Fatalf("tampered block #2 was marked valid!")
	}

	if resTampered.TamperedBlockIndex == nil || *resTampered.TamperedBlockIndex != 2 {
		t.Fatalf("expected tampered block index 2, got %v", resTampered.TamperedBlockIndex)
	}
}
