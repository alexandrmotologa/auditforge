package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWAL_WriteAndRead(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}
	defer wal.Close()

	signer, _ := NewEd25519Signer()

	// Write 5 blocks
	var prevHash []byte = make([]byte, 32)
	for i := uint64(0); i < 5; i++ {
		evts := []*AuditEvent{
			createTestEvent(NewEventID(), "item.create"),
			createTestEvent(NewEventID(), "item.update"),
		}
		b, _, err := NewBlock(i, prevHash, evts, AlgorithmBlake3)
		if err != nil {
			t.Fatalf("block creation failed: %v", err)
		}
		if err := b.Seal(signer); err != nil {
			t.Fatalf("block sealing failed: %v", err)
		}

		if _, err := wal.AppendBlock(b); err != nil {
			t.Fatalf("wal append failed: %v", err)
		}
		prevHash = b.Header.BlockHash
	}

	blocks, err := wal.ReadAllBlocks()
	if err != nil {
		t.Fatalf("wal read all blocks failed: %v", err)
	}

	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(blocks))
	}

	// Verify all read blocks
	res := VerifyLedgerChain(blocks)
	if !res.Valid {
		t.Fatalf("ledger chain verification failed: %s", res.Reason)
	}
}

func TestWAL_TamperDetectionOnDisk(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "tamper.wal")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}

	signer, _ := NewEd25519Signer()
	evts := []*AuditEvent{createTestEvent(NewEventID(), "sensitive.data.read")}
	b, _, _ := NewBlock(0, make([]byte, 32), evts, AlgorithmBlake3)
	_ = b.Seal(signer)
	_, _ = wal.AppendBlock(b)
	wal.Close()

	// Deliberately flip 1 byte inside the WAL file payload area
	data, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("failed reading wal file: %v", err)
	}

	// Byte offset within payload
	corruptOffset := 25
	data[corruptOffset] ^= 0x55
	if err := os.WriteFile(walPath, data, 0644); err != nil {
		t.Fatalf("failed writing tampered wal file: %v", err)
	}

	// Reopen WAL: must detect checksum failure or corrupt frame
	_, err = OpenWAL(walPath)
	if err == nil {
		t.Fatalf("tampered WAL opened without error!")
	}
}
