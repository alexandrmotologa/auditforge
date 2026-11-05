package export_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/export"
	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

func TestProofBundle_ExportAndVerify(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "export_test.wal")
	bundlePath := filepath.Join(tempDir, "compliance.proof")

	wal, err := ledger.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}
	defer wal.Close()

	signer, _ := ledger.NewEd25519Signer()

	// Write 3 sealed blocks
	var prevHash []byte = make([]byte, 32)
	for i := uint64(0); i < 3; i++ {
		evts := []*ledger.AuditEvent{
			{
				ID:        ledger.NewEventID(),
				Timestamp: time.Now().UTC(),
				TenantID:  "org_enterprise",
				Action:    "auth.mfa.verify",
				Actor:     ledger.Actor{ID: "usr_alice", Name: "Alice Engineer"},
				Target:    ledger.Target{ID: "app_portal", Type: "web_app"},
			},
			{
				ID:        ledger.NewEventID(),
				Timestamp: time.Now().UTC(),
				TenantID:  "org_enterprise",
				Action:    "data.query",
				Actor:     ledger.Actor{ID: "usr_alice", Name: "Alice Engineer"},
				Target:    ledger.Target{ID: "tbl_users", Type: "sql_table"},
			},
		}

		b, _, err := ledger.NewBlock(i, prevHash, evts, ledger.AlgorithmBlake3)
		if err != nil {
			t.Fatalf("failed creating block %d: %v", i, err)
		}
		if err := b.Seal(signer); err != nil {
			t.Fatalf("failed sealing block %d: %v", i, err)
		}
		if _, err := wal.AppendBlock(b); err != nil {
			t.Fatalf("failed appending block %d: %v", i, err)
		}
		prevHash = b.Header.BlockHash
	}

	// 1. Export proof bundle
	manifest, err := export.GenerateProofBundle(wal, 0, 2, signer, bundlePath)
	if err != nil {
		t.Fatalf("failed generating proof bundle: %v", err)
	}

	if manifest.TotalBlocks != 3 || manifest.TotalEvents != 6 {
		t.Fatalf("unexpected manifest counts: %+v", manifest)
	}

	// 2. Read proof bundle back
	readManifest, chain, events, err := export.ReadProofBundle(bundlePath)
	if err != nil {
		t.Fatalf("failed reading proof bundle: %v", err)
	}

	if readManifest.HeadBlockHash != manifest.HeadBlockHash {
		t.Fatalf("head block hash mismatch: %s vs %s", readManifest.HeadBlockHash, manifest.HeadBlockHash)
	}

	if len(chain) != 3 || len(events) != 6 {
		t.Fatalf("unexpected chain length (%d) or event count (%d)", len(chain), len(events))
	}

	// 3. Verify each event leaf and proof
	for _, pe := range events {
		calcLeaf, err := pe.Event.ComputeLeafHash(readManifest.Algorithm)
		if err != nil {
			t.Fatalf("leaf hash calculation failed: %v", err)
		}
		if !ledger.VerifyHash(calcLeaf, pe.Event.LeafHash) {
			t.Fatalf("leaf hash mismatch on event %s", pe.Event.ID)
		}
		if !ledger.VerifyMerkleProof(pe.Proof) {
			t.Fatalf("merkle inclusion proof verification failed for event %s", pe.Event.ID)
		}
	}
}
