package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/zeebo/blake3"
)

const (
	// AlgorithmBlake3 identifies the Blake3 cryptographic hash algorithm
	AlgorithmBlake3 = "blake3"
	// AlgorithmSHA256 identifies the SHA-256 cryptographic hash algorithm
	AlgorithmSHA256 = "sha256"

	// Domain separation prefixes (RFC 6962 / Merkle Tree security standard)
	leafPrefix byte = 0x00
	nodePrefix byte = 0x01
)

// NormalizeAlgorithm returns a standard lowercased algorithm name or defaults to blake3
func NormalizeAlgorithm(algo string) string {
	a := strings.ToLower(strings.TrimSpace(algo))
	if a == AlgorithmSHA256 {
		return AlgorithmSHA256
	}
	return AlgorithmBlake3
}

// HashBytes computes the raw cryptographic hash of the input bytes
func HashBytes(algorithm string, data []byte) []byte {
	algo := NormalizeAlgorithm(algorithm)
	if algo == AlgorithmSHA256 {
		h := sha256.Sum256(data)
		res := make([]byte, 32)
		copy(res, h[:])
		return res
	}

	// Blake3 default (32-byte digest)
	h := blake3.Sum256(data)
	res := make([]byte, 32)
	copy(res, h[:])
	return res
}

// HashHex computes the hash and returns it as a lowercase hex string
func HashHex(algorithm string, data []byte) string {
	return hex.EncodeToString(HashBytes(algorithm, data))
}

// HashLeaf computes the domain-separated leaf hash: Hash(0x00 || data)
func HashLeaf(algorithm string, data []byte) []byte {
	algo := NormalizeAlgorithm(algorithm)
	payload := make([]byte, 1+len(data))
	payload[0] = leafPrefix
	copy(payload[1:], data)

	if algo == AlgorithmSHA256 {
		h := sha256.Sum256(payload)
		res := make([]byte, 32)
		copy(res, h[:])
		return res
	}

	h := blake3.Sum256(payload)
	res := make([]byte, 32)
	copy(res, h[:])
	return res
}

// HashNode computes the domain-separated internal node hash: Hash(0x01 || left || right)
func HashNode(algorithm string, left, right []byte) []byte {
	algo := NormalizeAlgorithm(algorithm)
	payload := make([]byte, 1+len(left)+len(right))
	payload[0] = nodePrefix
	copy(payload[1:], left)
	copy(payload[1+len(left):], right)

	if algo == AlgorithmSHA256 {
		h := sha256.Sum256(payload)
		res := make([]byte, 32)
		copy(res, h[:])
		return res
	}

	h := blake3.Sum256(payload)
	res := make([]byte, 32)
	copy(res, h[:])
	return res
}

// VerifyHash matches calculated hash bytes with expected hex string
func VerifyHash(calculated []byte, expectedHex string) bool {
	return hex.EncodeToString(calculated) == strings.ToLower(strings.TrimSpace(expectedHex))
}

// FormatHashHex formats raw hash bytes as a lowercase hex string
func FormatHashHex(hash []byte) string {
	return hex.EncodeToString(hash)
}

// ParseHashHex parses a hex string into a 32-byte slice
func ParseHashHex(h string) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimSpace(h))
	if err != nil {
		return nil, fmt.Errorf("invalid hash hex string: %w", err)
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("expected 32 bytes for hash, got %d", len(b))
	}
	return b, nil
}
