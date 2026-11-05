package export

import (
	"archive/tar"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// ProofManifest defines the top-level metadata and signature envelope for a .proof archive
type ProofManifest struct {
	Version          string    `json:"version"`
	ExportedAt       time.Time `json:"exported_at"`
	Algorithm        string    `json:"algorithm"`
	StartBlockIndex  uint64    `json:"start_block_index"`
	EndBlockIndex    uint64    `json:"end_block_index"`
	TotalBlocks      uint64    `json:"total_blocks"`
	TotalEvents      uint64    `json:"total_events"`
	HeadBlockHash    string    `json:"head_block_hash"`
	GenesisBlockHash string    `json:"genesis_block_hash"`
	SignerKeyID      string    `json:"signer_key_id,omitempty"`
	SignerPublicKey  string    `json:"signer_public_key,omitempty"`
	Signature        string    `json:"signature,omitempty"`
}

// ProofPackageEvent wraps an audit event with its Merkle inclusion proof for offline verification
type ProofPackageEvent struct {
	Event *ledger.AuditEvent  `json:"event"`
	Proof *ledger.MerkleProof `json:"proof"`
}

// GenerateProofBundle packages a range of blocks and events into a signed .proof (tar.gz) archive
func GenerateProofBundle(wal *ledger.WAL, startBlock, endBlock uint64, signer ledger.Signer, outPath string) (*ProofManifest, error) {
	allBlocks, err := wal.ReadAllBlocks()
	if err != nil {
		return nil, fmt.Errorf("failed reading wal blocks: %w", err)
	}

	if len(allBlocks) == 0 {
		return nil, fmt.Errorf("ledger is empty; cannot generate proof bundle")
	}

	if endBlock >= uint64(len(allBlocks)) {
		endBlock = uint64(len(allBlocks) - 1)
	}
	if startBlock > endBlock {
		return nil, fmt.Errorf("start block %d cannot exceed end block %d", startBlock, endBlock)
	}

	selectedBlocks := allBlocks[startBlock : endBlock+1]
	var totalEvents uint64 = 0
	var headers []*ledger.BlockHeader
	var packagedEvents []ProofPackageEvent

	algo := selectedBlocks[0].Header.Algorithm

	for _, b := range selectedBlocks {
		headers = append(headers, &b.Header)
		totalEvents += uint64(len(b.Events))

		// Build Merkle tree for this block to extract inclusion proofs
		leaves := make([][]byte, len(b.Events))
		for i, e := range b.Events {
			lh, err := e.ComputeLeafHash(algo)
			if err != nil {
				return nil, fmt.Errorf("failed computing leaf hash for event %s: %w", e.ID, err)
			}
			leaves[i] = lh
		}

		tree := ledger.NewMerkleTree(algo, leaves)
		for i, e := range b.Events {
			proof, err := tree.GetProof(i)
			if err != nil {
				return nil, fmt.Errorf("failed generating proof for event %s: %w", e.ID, err)
			}
			packagedEvents = append(packagedEvents, ProofPackageEvent{
				Event: e,
				Proof: proof,
			})
		}
	}

	manifest := &ProofManifest{
		Version:          "1.0",
		ExportedAt:       time.Now().UTC(),
		Algorithm:        algo,
		StartBlockIndex:  startBlock,
		EndBlockIndex:    endBlock,
		TotalBlocks:      uint64(len(selectedBlocks)),
		TotalEvents:      totalEvents,
		HeadBlockHash:    selectedBlocks[len(selectedBlocks)-1].Header.BlockHashHex,
		GenesisBlockHash: selectedBlocks[0].Header.BlockHashHex,
	}

	// Sign manifest if signer provided
	if signer != nil {
		manifestPayload := fmt.Sprintf("%s:%d:%d:%s:%s",
			manifest.Algorithm, manifest.StartBlockIndex, manifest.EndBlockIndex,
			manifest.HeadBlockHash, manifest.ExportedAt.Format(time.RFC3339Nano))
		sig, err := signer.Sign([]byte(manifestPayload))
		if err == nil {
			manifest.SignerKeyID = signer.KeyID()
			manifest.SignerPublicKey = hex.EncodeToString(signer.PublicKey())
			manifest.Signature = hex.EncodeToString(sig)
		}
	}

	// Write tar.gz archive
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil && filepath.Dir(outPath) != "." {
		return nil, fmt.Errorf("failed creating output directory: %w", err)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed creating proof bundle file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Write manifest.json
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := writeTarFile(tw, "manifest.json", manifestBytes); err != nil {
		return nil, err
	}

	// 2. Write chain.json (Block headers)
	chainBytes, _ := json.MarshalIndent(headers, "", "  ")
	if err := writeTarFile(tw, "chain.json", chainBytes); err != nil {
		return nil, err
	}

	// 3. Write events.jsonl
	var eventsBuffer []byte
	for _, pe := range packagedEvents {
		b, _ := json.Marshal(pe)
		eventsBuffer = append(eventsBuffer, b...)
		eventsBuffer = append(eventsBuffer, '\n')
	}
	if err := writeTarFile(tw, "events.jsonl", eventsBuffer); err != nil {
		return nil, err
	}

	return manifest, nil
}

// ReadProofBundle extracts and parses all components of a .proof archive
func ReadProofBundle(bundlePath string) (*ProofManifest, []*ledger.BlockHeader, []ProofPackageEvent, error) {
	f, err := os.Open(bundlePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed opening proof file: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed decompressing proof bundle: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	var manifest *ProofManifest
	var chain []*ledger.BlockHeader
	var events []ProofPackageEvent

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("error reading tar archive: %w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed reading archive file %s: %w", header.Name, err)
		}

		switch header.Name {
		case "manifest.json":
			var m ProofManifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, nil, nil, fmt.Errorf("failed unmarshaling manifest.json: %w", err)
			}
			manifest = &m

		case "chain.json":
			var c []*ledger.BlockHeader
			if err := json.Unmarshal(data, &c); err != nil {
				return nil, nil, nil, fmt.Errorf("failed unmarshaling chain.json: %w", err)
			}
			chain = c

		case "events.jsonl":
			lines := splitLines(data)
			for _, line := range lines {
				if len(line) == 0 {
					continue
				}
				var pe ProofPackageEvent
				if err := json.Unmarshal(line, &pe); err != nil {
					return nil, nil, nil, fmt.Errorf("failed unmarshaling event in events.jsonl: %w", err)
				}
				events = append(events, pe)
			}
		}
	}

	if manifest == nil {
		return nil, nil, nil, fmt.Errorf("invalid bundle: missing manifest.json")
	}

	return manifest, chain, events, nil
}

func writeTarFile(tw *tar.Writer, name string, content []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0644,
		Size:    int64(len(content)),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed writing tar header for %s: %w", name, err)
	}
	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("failed writing tar content for %s: %w", name, err)
	}
	return nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	var start = 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
