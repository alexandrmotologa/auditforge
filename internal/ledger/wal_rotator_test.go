package ledger

import (
	"path/filepath"
	"testing"
)

func TestWALRotator_ArchiveSegment(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "active.wal")
	archiveDir := filepath.Join(tempDir, "archive")

	wal, err := OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}

	signer, _ := NewEd25519Signer()
	var prevHash []byte = make([]byte, 32)

	for i := uint64(0); i < 4; i++ {
		evts := []*AuditEvent{createTestEvent(NewEventID(), "archive.test")}
		b, _, _ := NewBlock(i, prevHash, evts, AlgorithmBlake3)
		_ = b.Seal(signer)
		_, _ = wal.AppendBlock(b)
		prevHash = b.Header.BlockHash
	}
	wal.Close()

	archivePath, manifest, err := ArchiveWALSegment(walPath, archiveDir, 1)
	if err != nil {
		t.Fatalf("archiving segment failed: %v", err)
	}

	if archivePath == "" || manifest == nil {
		t.Fatalf("expected valid archive path and manifest")
	}

	if manifest.TotalBlocks != 4 || manifest.StartBlock != 0 || manifest.EndBlock != 3 {
		t.Fatalf("manifest metadata mismatch: %+v", manifest)
	}
}
