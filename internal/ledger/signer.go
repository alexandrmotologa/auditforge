package ledger

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Signer signs block headers and proof bundle manifests
type Signer interface {
	Sign(data []byte) ([]byte, error)
	PublicKey() []byte
	KeyID() string
}

// Ed25519Signer implements Signer using Ed25519 asymmetric cryptography
type Ed25519Signer struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   string
}

// NewEd25519Signer creates a new ephemeral or memory-backed Ed25519 signer
func NewEd25519Signer() (*Ed25519Signer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 keypair: %w", err)
	}

	h := sha256.Sum256(pub)
	keyID := hex.EncodeToString(h[:8])

	return &Ed25519Signer{
		priv: priv,
		pub:  pub,
		id:   keyID,
	}, nil
}

// LoadOrCreateSigner loads an existing private key from disk or creates a new one
func LoadOrCreateSigner(keyPath string) (*Ed25519Signer, error) {
	if keyPath == "" {
		return NewEd25519Signer()
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			priv := ed25519.PrivateKey(data)
			pub := priv.Public().(ed25519.PublicKey)
			h := sha256.Sum256(pub)
			return &Ed25519Signer{
				priv: priv,
				pub:  pub,
				id:   hex.EncodeToString(h[:8]),
			}, nil
		}
	}

	// Generate new key
	signer, err := NewEd25519Signer()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}
	if err := os.WriteFile(keyPath, signer.priv, 0600); err != nil {
		return nil, fmt.Errorf("failed to persist private key: %w", err)
	}

	return signer, nil
}

// Sign signs the input bytes using the private key
func (s *Ed25519Signer) Sign(data []byte) ([]byte, error) {
	return ed25519.Sign(s.priv, data), nil
}

// PublicKey returns the 32-byte Ed25519 public key
func (s *Ed25519Signer) PublicKey() []byte {
	return []byte(s.pub)
}

// KeyID returns the 8-byte hex identifier for this public key
func (s *Ed25519Signer) KeyID() string {
	return s.id
}

// VerifySignature cryptographically verifies an Ed25519 signature
func VerifySignature(publicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize || len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}
