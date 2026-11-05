# auditforge

Tamper-evident, immutable audit log daemon and cryptographic compliance ledger engine. Built in Go with an embedded React 19 compliance explorer, an offline proof verification CLI, and client SDKs for Go and TypeScript.

Auditforge replaces mutable SQL tables and high-cost third-party audit SaaS by providing a self-hosted, append-only ledger backed by Blake3/SHA-256 Merkle trees, RFC 8785 canonical serialization, and cryptographic block sealing.

## Key Features

* **High-Throughput Ingestion**: Dual gRPC and REST HTTP ingestion endpoints with sub-millisecond write acknowledgement.
* **Cryptographic Merkle DAG**: Events are canonically serialized (RFC 8785) and hashed into binary Merkle trees. Blocks are chained using previous block hashes.
* **Strict Append-Only Storage**: Raw event data and block headers reside in a pure append-only Write-Ahead Log (WAL) with CRC32 frame checksums.
* **Role-Based Access Control (RBAC)**: Fine-grained API key scopes separating log emitters (`ingest_only`), compliance auditors (`auditor_read`), and system administrators (`admin`).
* **Real-Time SIEM Webhook Forwarding**: Asynchronously streams sealed audit blocks to external SIEM tools (Datadog, Splunk, Elastic) signed with HMAC-SHA256 (`X-AuditForge-Signature`).
* **Offline Proof Verification**: Standalone CLI (`auditverify`) verifies exported `.proof` packages without database credentials or network access.
* **Embedded Compliance UI**: React 19 dashboard compiled directly into the Go binary (`//go:embed`), featuring real-time Server-Sent Events (SSE) streaming, state-change visual diffs, and proof package generation.
* **External Timestamping (RFC 3161)**: Supports external TSA notary witnessing to anchor ledger head hashes against independent timestamping authorities.
* **WAL Segment Rotation & Archiving**: Command-line archiving (`auditforge archive`) compresses sealed segments into cold-storage `.wal.gz` files with cryptographic manifests.
* **First-Class SDKs**: Client libraries for Go (`sdk/go`) and TypeScript (`sdk/ts`) with automatic background batching and fallback retry buffers.

## Architecture Overview

```
Application Clients (Node, Python, Go, Java, cURL)
                     |
                     | gRPC / REST (POST /api/v1/events)
                     v
+--------------------------------------------------------+
|                   auditforge daemon                    |
|                                                        |
|  [ API Auth & Role-Based Access Control (RBAC) ]       |
|               |                                        |
|               v                                        |
|  [ RFC 8785 Canonical JSON Serialization ]             |
|               |                                        |
|               v                                        |
|  [ Lock-Free Batch Ring Buffer (100ms or 500 events) ] |
|               |                                        |
|               v                                        |
|  [ Merkle Tree Leaf Builder (Blake3 / SHA-256) ]       |
|               |                                        |
|               v                                        |
|  [ Block Sealer, Ed25519 Signer & Chain Linker ]       |
|         /                      |             \         |
|        v                       v              v        |
|  [ Append-Only WAL ]   [ SQLite Index ]   [ SIEM Webhooks ]
|  (Raw Ledger)          (Query & Filter)   (Splunk/Datadog)
|                                                        |
|  [ Real-Time SSE Stream & React 19 Compliance Cockpit] |
+--------------------------------------------------------+
```

## Quick Start

### 1. Build and Run

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/auditforge.git
cd auditforge

# Build server and offline verifier
go build -o bin/auditforge.exe ./cmd/auditforge
go build -o bin/auditverify.exe ./cmd/auditverify

# Start the daemon
./bin/auditforge.exe --config config.yaml
```

The server listens on:
* REST API: `http://localhost:8080`
* gRPC Ingestion: `localhost:9090`
* Web Compliance Explorer: `http://localhost:8080/ui/`

### 2. Ingest an Audit Event (REST)

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Content-Type: application/json" \
  -H "X-API-Key: default-dev-key" \
  -d '{
    "tenant_id": "org_enterprise",
    "actor": {
      "id": "usr_alexander",
      "name": "Alexandr Motologa",
      "email": "alexandr@example.com",
      "role": "admin"
    },
    "action": "user.permission.grant",
    "target": {
      "type": "database_credential",
      "id": "cred_prod_readonly",
      "name": "Production Read-Only Cluster"
    },
    "metadata": {
      "ip_address": "192.168.1.100",
      "previous_state": { "role": "viewer" },
      "new_state": { "role": "admin" }
    }
  }'
```

Response:
```json
{
  "status": "accepted",
  "event_id": "evt_01JBCDEFGH890123456789",
  "leaf_hash": "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e",
  "received_at": "2026-11-05T19:35:00Z"
}
```

### 3. Client SDK Usage

#### Go SDK (`sdk/go`)

```go
package main

import (
    "context"
    "time"
    auditforge "github.com/alexandrmotologa/auditforge/sdk/go"
)

func main() {
    client := auditforge.NewClient(auditforge.Config{
        Endpoint: "http://localhost:8080",
        APIKey:   "your-ingest-api-key",
        TenantID: "org_enterprise",
    })
    defer client.Close()

    // Non-blocking asynchronous emit (buffered in memory)
    client.Emit(&auditforge.Event{
        Action: "role.grant",
        Actor:  auditforge.Actor{ID: "usr_alice", Name: "Alice Engineer"},
        Target: auditforge.Target{ID: "cred_db", Type: "database"},
    })
}
```

#### TypeScript / Node SDK (`sdk/ts`)

```typescript
import { AuditForgeClient } from '@auditforge/client';

const client = new AuditForgeClient({
  endpoint: 'http://localhost:8080',
  apiKey: 'your-ingest-api-key',
  tenantId: 'org_enterprise',
});

// Non-blocking background batch emission
client.emit({
  action: 'data.export',
  actor: { id: 'usr_bob', name: 'Bob Auditor' },
  target: { id: 'rep_q3', type: 'report' },
});
```

### 4. Verify Ledger Integrity Offline

Export a proof package from the web UI or CLI:
```bash
./bin/auditforge.exe export-proof --from 0 --to 100 --out export.proof
```

Verify with zero dependencies:
```bash
./bin/auditverify.exe verify export.proof
```

Verification output:
```text
[OK] Loaded proof bundle: export.proof
[OK] Blocks verified: 50 (from Block #0 to Block #50)
[OK] Total events validated: 12,480
[OK] Merkle root consistency: 100% VALID
[OK] Cryptographic chain link: UNBROKEN
RESULT: LEDGER INTEGRITY CONFIRMED (0 TAMPER DETECTED)
```

### 5. Archive WAL Segments

Compress and seal inactive segments for cold-storage backup:
```bash
./bin/auditforge.exe archive --segment 1 --out ./data/archive
```

## Proof Bundle Format (`.proof`)

An auditforge `.proof` archive contains:
1. `manifest.json`: Metadata, algorithm version, signing key ID, Ed25519 digital signature, and root hash summary.
2. `chain.json`: Ordered list of block headers, previous block hashes, Merkle roots, timestamps, and block signatures.
3. `events.jsonl`: Canonical JSON events with individual Merkle audit paths (sibling hashes).

## License

MIT License. See [LICENSE](LICENSE) for details.
