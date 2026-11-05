package ledger

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// SegmentManifest documents an archived immutable WAL segment
type SegmentManifest struct {
	SegmentIndex uint64    `json:"segment_index"`
	CreatedAt    time.Time `json:"created_at"`
	StartBlock   uint64    `json:"start_block"`
	EndBlock     uint64    `json:"end_block"`
	TotalBlocks  uint64    `json:"total_blocks"`
	StartHash    string    `json:"start_hash"`
	EndHash      string    `json:"end_hash"`
	ArchiveSHA256 string   `json:"archive_sha256"`
}

// ArchiveWALSegment compresses an existing WAL file into an immutable .wal.gz segment with signed manifest
func ArchiveWALSegment(walPath, archiveDir string, segmentIndex uint64) (string, *SegmentManifest, error) {
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed creating archive directory: %w", err)
	}

	wal, err := OpenWAL(walPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed opening wal for archiving: %w", err)
	}

	blocks, err := wal.ReadAllBlocks()
	wal.Close()
	if err != nil {
		return "", nil, fmt.Errorf("failed reading blocks for archiving: %w", err)
	}

	if len(blocks) == 0 {
		return "", nil, fmt.Errorf("wal is empty; nothing to archive")
	}

	startBlock := blocks[0].Header.Index
	endBlock := blocks[len(blocks)-1].Header.Index
	startHash := blocks[0].Header.BlockHashHex
	endHash := blocks[len(blocks)-1].Header.BlockHashHex

	archiveName := fmt.Sprintf("segment_%05d.wal.gz", segmentIndex)
	archivePath := filepath.Join(archiveDir, archiveName)

	srcFile, err := os.Open(walPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed opening source wal file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(archivePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed creating archive file: %w", err)
	}
	defer destFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(destFile, hasher)

	gw := gzip.NewWriter(multiWriter)
	if _, err := io.Copy(gw, srcFile); err != nil {
		gw.Close()
		return "", nil, fmt.Errorf("failed compressing wal segment: %w", err)
	}
	if err := gw.Close(); err != nil {
		return "", nil, fmt.Errorf("failed closing gzip writer: %w", err)
	}

	archiveSHA256 := hex.EncodeToString(hasher.Sum(nil))

	manifest := &SegmentManifest{
		SegmentIndex:  segmentIndex,
		CreatedAt:     time.Now().UTC(),
		StartBlock:    startBlock,
		EndBlock:      endBlock,
		TotalBlocks:   uint64(len(blocks)),
		StartHash:     startHash,
		EndHash:       endHash,
		ArchiveSHA256: archiveSHA256,
	}

	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	manifestPath := filepath.Join(archiveDir, fmt.Sprintf("segment_%05d.manifest.json", segmentIndex))
	if err := os.WriteFile(manifestPath, manifestBytes, 0644); err != nil {
		return "", nil, fmt.Errorf("failed writing segment manifest: %w", err)
	}

	return archivePath, manifest, nil
}
