package ledger

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
)

// ProofPosition designates whether a sibling node lies to the left or right of the path
type ProofPosition string

const (
	PositionLeft  ProofPosition = "left"
	PositionRight ProofPosition = "right"
)

// MerkleProofStep stores an individual sibling hash and its relative position in the tree
type MerkleProofStep struct {
	Position    ProofPosition `json:"position"`
	SiblingHash []byte        `json:"sibling_hash"`
	SiblingHex  string        `json:"sibling_hex"`
}

// MerkleProof contains all cryptographic sibling hashes necessary to independently verify
// that an audit event leaf belongs to the certified Merkle root.
type MerkleProof struct {
	LeafIndex int               `json:"leaf_index"`
	LeafHash  []byte            `json:"leaf_hash"`
	LeafHex   string            `json:"leaf_hex"`
	AuditPath []MerkleProofStep `json:"audit_path"`
	RootHash  []byte            `json:"root_hash"`
	RootHex   string            `json:"root_hex"`
	Algorithm string            `json:"algorithm"`
}

// MerkleTree represents a balanced binary Merkle tree calculated over a set of leaves
type MerkleTree struct {
	Algorithm string
	Leaves    [][]byte
	Levels    [][][]byte
	Root      []byte
}

// NewMerkleTree constructs a complete binary Merkle tree from raw leaf hashes
func NewMerkleTree(algorithm string, leaves [][]byte) *MerkleTree {
	algo := NormalizeAlgorithm(algorithm)
	if len(leaves) == 0 {
		emptyRoot := HashBytes(algo, []byte(""))
		return &MerkleTree{
			Algorithm: algo,
			Leaves:    leaves,
			Levels:    [][][]byte{{emptyRoot}},
			Root:      emptyRoot,
		}
	}

	// Copy leaves to prevent external mutation
	currentLevel := make([][]byte, len(leaves))
	for i, l := range leaves {
		leafCopy := make([]byte, len(l))
		copy(leafCopy, l)
		currentLevel[i] = leafCopy
	}

	levels := [][][]byte{currentLevel}

	// Special case: single leaf. Root is HashNode(leaf, leaf) to maintain depth
	if len(currentLevel) == 1 {
		root := HashNode(algo, currentLevel[0], currentLevel[0])
		levels = append(levels, [][]byte{root})
		return &MerkleTree{
			Algorithm: algo,
			Leaves:    currentLevel,
			Levels:    levels,
			Root:      root,
		}
	}

	// Build tree levels bottom-up
	for len(currentLevel) > 1 {
		var nextLevel [][]byte
		for i := 0; i < len(currentLevel); i += 2 {
			left := currentLevel[i]
			var right []byte
			if i+1 < len(currentLevel) {
				right = currentLevel[i+1]
			} else {
				// Odd number of nodes: duplicate last node (RFC 6962 / Bitcoin standard)
				right = currentLevel[i]
			}
			parent := HashNode(algo, left, right)
			nextLevel = append(nextLevel, parent)
		}
		levels = append(levels, nextLevel)
		currentLevel = nextLevel
	}

	return &MerkleTree{
		Algorithm: algo,
		Leaves:    leaves,
		Levels:    levels,
		Root:      currentLevel[0],
	}
}

// GetProof extracts the cryptographic audit path for the leaf at leafIndex
func (m *MerkleTree) GetProof(leafIndex int) (*MerkleProof, error) {
	if leafIndex < 0 || leafIndex >= len(m.Leaves) {
		return nil, fmt.Errorf("leaf index %d out of bounds (total leaves: %d)", leafIndex, len(m.Leaves))
	}

	var steps []MerkleProofStep
	currentIndex := leafIndex

	// Iterate through levels from leaves up to the level below root
	for levelIdx := 0; levelIdx < len(m.Levels)-1; levelIdx++ {
		currentLevel := m.Levels[levelIdx]
		var sibling []byte
		var position ProofPosition

		if currentIndex%2 == 0 {
			// Current node is on the left; sibling is on the right
			position = PositionRight
			if currentIndex+1 < len(currentLevel) {
				sibling = currentLevel[currentIndex+1]
			} else {
				// Duplicated node scenario
				sibling = currentLevel[currentIndex]
			}
		} else {
			// Current node is on the right; sibling is on the left
			position = PositionLeft
			sibling = currentLevel[currentIndex-1]
		}

		siblingCopy := make([]byte, len(sibling))
		copy(siblingCopy, sibling)

		steps = append(steps, MerkleProofStep{
			Position:    position,
			SiblingHash: siblingCopy,
			SiblingHex:  hex.EncodeToString(siblingCopy),
		})

		// Move up to parent index
		currentIndex /= 2
	}

	targetLeaf := m.Leaves[leafIndex]
	leafCopy := make([]byte, len(targetLeaf))
	copy(leafCopy, targetLeaf)

	rootCopy := make([]byte, len(m.Root))
	copy(rootCopy, m.Root)

	return &MerkleProof{
		LeafIndex: leafIndex,
		LeafHash:  leafCopy,
		LeafHex:   hex.EncodeToString(leafCopy),
		AuditPath: steps,
		RootHash:  rootCopy,
		RootHex:   hex.EncodeToString(rootCopy),
		Algorithm: m.Algorithm,
	}, nil
}

// VerifyMerkleProof cryptographically checks that leafHash recalculates to expected root via the audit path
func VerifyMerkleProof(proof *MerkleProof) bool {
	if proof == nil || len(proof.LeafHash) == 0 || len(proof.RootHash) == 0 {
		return false
	}

	algo := NormalizeAlgorithm(proof.Algorithm)
	current := make([]byte, len(proof.LeafHash))
	copy(current, proof.LeafHash)

	for _, step := range proof.AuditPath {
		if step.Position == PositionRight {
			current = HashNode(algo, current, step.SiblingHash)
		} else if step.Position == PositionLeft {
			current = HashNode(algo, step.SiblingHash, current)
		} else {
			return false
		}
	}

	return bytes.Equal(current, proof.RootHash)
}

// VerifyLeafAgainstRoot verifies an audit proof against a target expected root byte slice
func VerifyLeafAgainstRoot(algorithm string, leafHash []byte, path []MerkleProofStep, expectedRoot []byte) (bool, error) {
	if len(leafHash) == 0 || len(expectedRoot) == 0 {
		return false, errors.New("empty leaf or root hash provided")
	}

	algo := NormalizeAlgorithm(algorithm)
	current := make([]byte, len(leafHash))
	copy(current, leafHash)

	for _, step := range path {
		if step.Position == PositionRight {
			current = HashNode(algo, current, step.SiblingHash)
		} else if step.Position == PositionLeft {
			current = HashNode(algo, step.SiblingHash, current)
		} else {
			return false, fmt.Errorf("invalid step position: %s", step.Position)
		}
	}

	return bytes.Equal(current, expectedRoot), nil
}
