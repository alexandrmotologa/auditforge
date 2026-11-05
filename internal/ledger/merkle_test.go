package ledger

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMerkleTree_Proofs(t *testing.T) {
	algorithms := []string{AlgorithmBlake3, AlgorithmSHA256}
	counts := []int{1, 2, 3, 4, 5, 8, 17, 32}

	for _, algo := range algorithms {
		t.Run(algo, func(t *testing.T) {
			for _, count := range counts {
				t.Run(fmt.Sprintf("%d_leaves", count), func(t *testing.T) {
					leaves := make([][]byte, count)
					for i := 0; i < count; i++ {
						leaves[i] = HashLeaf(algo, []byte(fmt.Sprintf("event-data-%d", i)))
					}

					tree := NewMerkleTree(algo, leaves)
					if len(tree.Root) != 32 {
						t.Fatalf("expected 32-byte root, got %d", len(tree.Root))
					}

					// Verify inclusion proof for each individual leaf
					for i := 0; i < count; i++ {
						proof, err := tree.GetProof(i)
						if err != nil {
							t.Fatalf("failed getting proof for leaf %d: %v", i, err)
						}

						if !VerifyMerkleProof(proof) {
							t.Fatalf("failed verifying valid proof for leaf %d", i)
						}

						// Tamper with leaf and ensure proof verification FAILS
						tamperedProof := *proof
						tamperedProof.LeafHash = append([]byte(nil), proof.LeafHash...)
						tamperedProof.LeafHash[0] ^= 0xFF // Flip bits

						if VerifyMerkleProof(&tamperedProof) {
							t.Fatalf("proof verification succeeded on tampered leaf %d", i)
						}
					}
				})
			}
		})
	}
}

func TestMerkleTree_DeterministicRoot(t *testing.T) {
	leaves := [][]byte{
		HashLeaf(AlgorithmBlake3, []byte("leaf-1")),
		HashLeaf(AlgorithmBlake3, []byte("leaf-2")),
		HashLeaf(AlgorithmBlake3, []byte("leaf-3")),
	}

	tree1 := NewMerkleTree(AlgorithmBlake3, leaves)
	tree2 := NewMerkleTree(AlgorithmBlake3, leaves)

	if !bytes.Equal(tree1.Root, tree2.Root) {
		t.Fatalf("merkle root non-deterministic between identical runs")
	}
}
