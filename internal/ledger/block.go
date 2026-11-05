package ledger

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// BlockHeader defines the cryptographic seal and metadata for a block of audit events
type BlockHeader struct {
	Index        uint64    `json:"index"`
	PreviousHash []byte    `json:"previous_hash"`
	PreviousHex  string    `json:"previous_hex"`
	MerkleRoot   []byte    `json:"merkle_root"`
	RootHex      string    `json:"root_hex"`
	Timestamp    time.Time `json:"timestamp"`
	EventCount   uint32    `json:"event_count"`
	Algorithm    string    `json:"algorithm"`
	Signature    []byte    `json:"signature,omitempty"`
	PublicKey    []byte    `json:"public_key,omitempty"`
	KeyID        string    `json:"key_id,omitempty"`
	BlockHash    []byte    `json:"block_hash"`
	BlockHashHex string    `json:"block_hash_hex"`
}

// Block represents a sealed cryptographic container holding an ordered list of audit events
type Block struct {
	Header BlockHeader   `json:"header"`
	Events []*AuditEvent `json:"events"`
}

// HeaderPayload defines the canonical structure hashed to produce BlockHash
type HeaderPayload struct {
	Algorithm    string `json:"algorithm"`
	EventCount   uint32 `json:"event_count"`
	Index        uint64 `json:"index"`
	MerkleRoot   string `json:"merkle_root"`
	PreviousHash string `json:"previous_hash"`
	Timestamp    string `json:"timestamp"`
}

// NewBlock constructs a block from events, builds the Merkle tree, and prepares the header
func NewBlock(index uint64, previousHash []byte, events []*AuditEvent, algorithm string) (*Block, *MerkleTree, error) {
	algo := NormalizeAlgorithm(algorithm)
	if previousHash == nil {
		previousHash = make([]byte, 32) // Genesis block previous hash is 32 zero bytes
	}

	leaves := make([][]byte, len(events))
	for i, evt := range events {
		evt.BlockIndex = index
		evt.LeafIndex = i
		leafHash, err := evt.ComputeLeafHash(algo)
		if err != nil {
			return nil, nil, fmt.Errorf("failed computing leaf hash for event %s: %w", evt.ID, err)
		}
		leaves[i] = leafHash
	}

	tree := NewMerkleTree(algo, leaves)

	header := BlockHeader{
		Index:        index,
		PreviousHash: previousHash,
		PreviousHex:  hex.EncodeToString(previousHash),
		MerkleRoot:   tree.Root,
		RootHex:      hex.EncodeToString(tree.Root),
		Timestamp:    time.Now().UTC(),
		EventCount:   uint32(len(events)),
		Algorithm:    algo,
	}

	block := &Block{
		Header: header,
		Events: events,
	}

	return block, tree, nil
}

// ComputeHeaderHash deterministically hashes the block header fields
func (b *Block) ComputeHeaderHash() ([]byte, error) {
	ts := b.Header.Timestamp.UTC().Format(time.RFC3339Nano)
	payload := HeaderPayload{
		Algorithm:    b.Header.Algorithm,
		EventCount:   b.Header.EventCount,
		Index:        b.Header.Index,
		MerkleRoot:   b.Header.RootHex,
		PreviousHash: b.Header.PreviousHex,
		Timestamp:    ts,
	}

	canonicalBytes, err := CanonicalizeJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize block header: %w", err)
	}

	return HashBytes(b.Header.Algorithm, canonicalBytes), nil
}

// Seal seals the block by computing its block hash and optionally signing it
func (b *Block) Seal(signer Signer) error {
	blockHash, err := b.ComputeHeaderHash()
	if err != nil {
		return err
	}

	b.Header.BlockHash = blockHash
	b.Header.BlockHashHex = hex.EncodeToString(blockHash)

	if signer != nil {
		sig, err := signer.Sign(blockHash)
		if err != nil {
			return fmt.Errorf("failed to sign block %d: %w", b.Header.Index, err)
		}
		b.Header.Signature = sig
		b.Header.PublicKey = signer.PublicKey()
		b.Header.KeyID = signer.KeyID()
	}

	return nil
}

// VerifyIntegrity checks internal cryptographic consistency of this block:
// 1. Every event matches its leaf hash
// 2. Merkle tree calculated over events matches Header.MerkleRoot
// 3. Header hash recalculates to Header.BlockHash
// 4. Ed25519 signature is valid if present
func (b *Block) VerifyIntegrity() (bool, error) {
	algo := NormalizeAlgorithm(b.Header.Algorithm)

	if len(b.Events) != int(b.Header.EventCount) {
		return false, fmt.Errorf("event count mismatch: header says %d, block has %d", b.Header.EventCount, len(b.Events))
	}

	leaves := make([][]byte, len(b.Events))
	for i, evt := range b.Events {
		leafHash, err := evt.ComputeLeafHash(algo)
		if err != nil {
			return false, fmt.Errorf("event %s leaf hash recomputation failed: %w", evt.ID, err)
		}
		if evt.LeafHash != hex.EncodeToString(leafHash) {
			return false, fmt.Errorf("event %s leaf hash mismatch", evt.ID)
		}
		leaves[i] = leafHash
	}

	tree := NewMerkleTree(algo, leaves)
	if !bytes.Equal(tree.Root, b.Header.MerkleRoot) {
		return false, fmt.Errorf("merkle root mismatch: calculated %x, header has %x", tree.Root, b.Header.MerkleRoot)
	}

	calcBlockHash, err := b.ComputeHeaderHash()
	if err != nil {
		return false, fmt.Errorf("block header hash recomputation failed: %w", err)
	}

	if !bytes.Equal(calcBlockHash, b.Header.BlockHash) {
		return false, fmt.Errorf("block hash mismatch: calculated %x, header has %x", calcBlockHash, b.Header.BlockHash)
	}

	if len(b.Header.Signature) > 0 && len(b.Header.PublicKey) > 0 {
		if !VerifySignature(b.Header.PublicKey, b.Header.BlockHash, b.Header.Signature) {
			return false, errors.New("invalid cryptographic block signature")
		}
	}

	return true, nil
}
