package ledger

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
)

// VerificationResult contains the comprehensive outcome of an audit ledger verification
type VerificationResult struct {
	Valid              bool          `json:"valid"`
	TotalBlocks        uint64        `json:"total_blocks"`
	TotalEvents        uint64        `json:"total_events"`
	TamperedBlockIndex *uint64       `json:"tampered_block_index,omitempty"`
	Reason             string        `json:"reason,omitempty"`
	Duration           time.Duration `json:"duration_ms"`
	HeadBlockHash      string        `json:"head_block_hash,omitempty"`
	GenesisBlockHash   string        `json:"genesis_block_hash,omitempty"`
	VerifiedAt         time.Time     `json:"verified_at"`
}

// VerifyLedgerChain comprehensively verifies the entire cryptographic chain of blocks from genesis to head
func VerifyLedgerChain(blocks []*Block) *VerificationResult {
	start := time.Now()
	res := &VerificationResult{
		Valid:      true,
		VerifiedAt: start.UTC(),
	}

	if len(blocks) == 0 {
		res.Duration = time.Since(start)
		return res
	}

	var totalEvents uint64 = 0
	var prevHash []byte = make([]byte, 32) // Genesis block expects 32 zero bytes

	for i, b := range blocks {
		idx := b.Header.Index

		// 1. Verify sequence index
		if idx != uint64(i) {
			res.Valid = false
			res.TamperedBlockIndex = &idx
			res.Reason = fmt.Sprintf("block sequence violation: expected index %d, found %d", i, idx)
			res.Duration = time.Since(start)
			return res
		}

		// 2. Verify previous block hash link
		if !bytes.Equal(b.Header.PreviousHash, prevHash) {
			res.Valid = false
			res.TamperedBlockIndex = &idx
			res.Reason = fmt.Sprintf("broken hash chain at block %d: previous hash %x does not match predecessor hash %x",
				idx, b.Header.PreviousHash, prevHash)
			res.Duration = time.Since(start)
			return res
		}

		// 3. Verify internal block integrity (events, leaves, Merkle root, block hash, signature)
		if valid, err := b.VerifyIntegrity(); !valid {
			res.Valid = false
			res.TamperedBlockIndex = &idx
			res.Reason = fmt.Sprintf("block %d internal integrity check failed: %v", idx, err)
			res.Duration = time.Since(start)
			return res
		}

		// 4. Verify timestamp monotonicity (block k timestamp should not precede block k-1)
		if i > 0 && b.Header.Timestamp.Before(blocks[i-1].Header.Timestamp) {
			res.Valid = false
			res.TamperedBlockIndex = &idx
			res.Reason = fmt.Sprintf("timestamp regression at block %d: %s is before predecessor %s",
				idx, b.Header.Timestamp.Format(time.RFC3339Nano), blocks[i-1].Header.Timestamp.Format(time.RFC3339Nano))
			res.Duration = time.Since(start)
			return res
		}

		totalEvents += uint64(b.Header.EventCount)
		prevHash = b.Header.BlockHash
	}

	res.TotalBlocks = uint64(len(blocks))
	res.TotalEvents = totalEvents
	res.GenesisBlockHash = hex.EncodeToString(blocks[0].Header.BlockHash)
	res.HeadBlockHash = hex.EncodeToString(blocks[len(blocks)-1].Header.BlockHash)
	res.Duration = time.Since(start)

	return res
}
