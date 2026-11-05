# auditforge

Tamper-evident, immutable audit log daemon and cryptographic compliance ledger engine. Built in Go with an embedded React 19 compliance explorer and an offline proof verification CLI.

Auditforge replaces mutable SQL tables and high-cost third-party audit SaaS by providing a self-hosted, append-only ledger backed by Blake3/SHA-256 Merkle trees and cryptographic block sealing.

## Key Features

* **High-Throughput Ingestion**: Dual gRPC and REST HTTP ingestion endpoints with sub-millisecond write acknowledgement.
* **Cryptographic Merkle DAG**: Events are canonically serialized (RFC 8785) and hashed into binary Merkle trees. Blocks are chained using previous block hashes.
* **Strict Append-Only Storage**: Raw event data and block headers reside in a pure append-only Write-Ahead Log (WAL). Zero in-place modifications.
* **Offline Proof Verification**: Standalone CLI (`auditverify`) verifies exported `.proof` packages without database credentials or network access.
* **Fast Query Engine**: Embedded SQLite database indexes actors, actions, targets, and timestamps without mutating original log blocks.
* **Embedded Compliance UI**: React 19 dashboard compiled directly into the Go binary (`//go:embed`), featuring real-time Server-Sent Events (SSE) streaming and proof package generation.
* **SOC 2 Type II and HIPAA Ready**: Deterministic hash trees satisfy auditor requirements for data integrity and non-repudiation.

## Architecture Overview

```
Application Clients (Node, Python, Go, Java, cURL)
                     |
                     | gRPC / REST (POST /api/v1/events)
                     v
+--------------------------------------------------------+
|                   auditforge daemon                    |
|                                                        |
|  [ API Auth & Rate Limiter ]                           |
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
|  [ Block Sealer & Cryptographic Chain Linker ]         |
|         /                                    \         |
|        v                                      v        |
|  [ Append-Only WAL ]                  [ SQLite Index ] |
|  (Raw Cryptographic Ledger)           (Query & Filter) |
|                                                        |
|  [ Real-Time SSE Stream & React 19 Compliance Cockpit] |
+--------------------------------------------------------+
```

## Quick Start

### 1. Run via Go

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
* gRPC: `localhost:9090`
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
      "session_id": "sess_89f72b1a"
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

### 3. Verify Ledger Integrity Offline

Export a proof package from the web UI or CLI:
```bash
./bin/auditforge.exe export-proof --block-from 1 --block-to 50 --out export.proof
```

Verify with zero dependencies:
```bash
./bin/auditverify.exe verify export.proof
```

Verification output:
```text
[OK] Loaded proof bundle: export.proof
[OK] Blocks verified: 50 (from Block #1 to Block #50)
[OK] Total events validated: 12,480
[OK] Merkle root consistency: 100% VALID
[OK] Cryptographic chain link: UNBROKEN
RESULT: LEDGER INTEGRITY CONFIRMED (0 TAMPER DETECTED)
```

## Proof Bundle Format (`.proof`)

An auditforge `.proof` archive contains:
1. `manifest.json`: Metadata, algorithm version, signing key ID, and root hash summary.
2. `chain.json`: Ordered list of block headers, previous block hashes, Merkle roots, timestamps, and block signatures.
3. `events.jsonl`: Canonical JSON events with individual Merkle audit paths (sibling hashes).

## License

MIT License. See [LICENSE](LICENSE) for details.
