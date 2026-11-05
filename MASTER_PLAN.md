# `auditforge` — Architecture & Engineering Master Plan

> **Tamper-Evident Immutable Audit Log & Cryptographic Compliance Ledger Engine (SOC 2 Type II, ISO 27001 & HIPAA Ready)**  
> *Self-Hosted WorkOS Audit Logs and Retool Ledger Alternative in a Single Binary (Go + React 19)*

---

## 1. Executive Summary & Problem Space

### The Core Problem
To close enterprise B2B sales contracts and satisfy regulatory compliance standards (SOC 2 Type II, ISO 27001, HIPAA, GDPR), software applications must provide immutable, tamper-evident audit trails. These logs must capture every critical state change, permission mutation, data export, and authentication attempt.

Existing solutions present severe drawbacks:
1. **Third-Party SaaS Vendors (WorkOS Audit Logs, Retool Ledger, Datadog Audit)**: Prohibitively expensive (\$300–\$1,000+/month per organization), leak sensitive customer metadata outside your security perimeter, and create vendor lock-in.
2. **Homemade SQL Tables (`audit_logs`)**: Fundamentally flawed from a compliance perspective. Because database administrators (DBAs) or compromised server credentials can execute `UPDATE` or `DELETE` queries on relational tables, compliance auditors routinely flag them as *mutable* and non-compliant.

### The Solution: `auditforge`
`auditforge` is a self-hosted, lightweight, single-binary compliance audit log daemon. Applications ingest structured audit events via high-throughput gRPC or REST APIs. Events are sealed into cryptographic Merkle DAG blocks (SHA-256 / Blake3 hash chains) backed by an append-only Write-Ahead Log (WAL). If a single byte on disk is altered or deleted, the cryptographic verification fails immediately.

Auditors can verify log integrity offline with a zero-dependency CLI command (`auditforge verify export.proof`) without needing network access or database credentials.

---

## 2. System Architecture & Cryptographic Proof Chain

```
[ Application Services (Node, Python, Go, Java, Ruby) ]
                       │
                       │  gRPC / REST: POST /api/v1/events
                       ▼
┌────────────────────────────────────────────────────────────────────────┐
│                        `auditforge` Core Engine                        │
│                                                                        │
│  ┌───────────────────────┐         ┌────────────────────────────────┐  │
│  │ Tenant & API Key Auth │ ──────► │ JSON Schema Event Validator    │  │
│  └───────────────────────┘         └────────────────┬───────────────┘  │
│                                                     │                  │
│                                                     ▼                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ In-Memory Lock-Free Ring Buffer (RingBuffer Batch Accumulator)   │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │ Every 100ms or 500 events        │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Merkle Tree Leaf Builder (Blake3 / SHA-256 Canonical Hashing)    │  │
│  │   Leaf_i = Hash(CanonicalJSON(Event_i))                          │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│                                     ▼                                  │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Cryptographic Block Sealer & Chain Linker                        │  │
│  │   Block_N = MerkleRoot_N + Hash(Block_{N-1}) + Timestamp + Nonce │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│         ┌───────────────────────────┴──────────────────────────┐       │
│         ▼                                                      ▼       │
│  ┌───────────────────────────────┐           ┌──────────────────────┐  │
│  │ Append-Only WAL & Merkle Tree │           │ SQLite Search Index  │  │
│  │ (Raw Tamper-Evident Ledger)   │           │ (Fast Query & Filter)│  │
│  └───────────────────────────────┘           └──────────────────────┘  │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Real-Time SSE Streamer & Embedded React 19 Compliance Explorer   │  │
│  │ - 1-Click SOC 2 / HIPAA Cryptographic Proof Bundle Export (.proof)│  │
│  │ - Instant Visual Tamper Detection Shield                         │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technology Stack

| Layer | Component | Specification |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.23+ | Ultra-fast native execution, deterministic memory management, zero CGo requirements |
| **Transport Protocols** | REST + gRPC | Dual-protocol ingestion for maximum framework interoperability |
| **Cryptographic Primitives** | Blake3 & SHA-256 | High-throughput hashing (> 3 GB/s per core on AVX-512) for leaf nodes and Merkle roots |
| **Data Immutability Engine** | Append-Only Block WAL | Strictly append-only sequential file storage with zero in-place updates |
| **Search & Filtering Engine** | Embedded SQLite 3.45+ | Fast multi-index querying (actor, action, target, timestamp, IP) without mutating raw blocks |
| **Embedded Web UI** | React 19 + TypeScript + Vite | Packaged directly inside the Go binary via `//go:embed` |
| **Styling & Components** | Tailwind CSS + Lucide Icons | Enterprise-grade, clean dark-mode compliance cockpit |

---

## 4. Repository & Directory Structure

```
B:\workgit\auditforge\
├── cmd\
│   ├── auditforge\
│   │   └── main.go                 # Server daemon entrypoint (REST, gRPC, Web UI)
│   └── auditverify\
│       └── main.go                 # Zero-dependency offline cryptographic proof validator CLI
├── internal\
│   ├── config\
│   │   └── config.go               # Configuration (storage path, port, retention, TLS)
│   ├── api\
│   │   ├── rest\
│   │   │   ├── handler.go          # HTTP REST endpoints (/api/v1/events, /api/v1/verify)
│   │   │   └── sse.go              # Server-Sent Events live audit stream
│   │   └── grpc\
│   │       ├── audit.proto         # Protocol Buffers definition for high-speed ingestion
│   │       └── server.go           # gRPC server implementation
│   ├── ledger\
│   │   ├── event.go                # AuditEvent data model and canonical serialization
│   │   ├── merkle.go               # Merkle Tree calculation and leaf hash generator
│   │   ├── block.go                # Cryptographic block structures and hash chain linker
│   │   ├── wal.go                  # Append-only WAL writer with crash recovery
│   │   └── verifier.go             # Cryptographic proof generation and chain integrity verifier
│   ├── index\
│   │   ├── sqlite.go               # Search index database connector and migrations
│   │   └── query.go                # Multi-field indexed search and pagination
│   └── export\
│       ├── proof_bundle.go         # Signed `.proof` export packaging (tar/gz + root signature)
│       └── report_pdf.go           # Structured SOC 2 compliance summary report generator
├── web\
│   ├── src\
│   │   ├── components\
│   │   │   ├── AuditFeed.tsx       # Real-time streaming audit event log viewer
│   │   │   ├── EventDetailModal.tsx# Deep inspector with canonical JSON and Merkle leaf hash
│   │   │   ├── IntegrityBadge.tsx  # Dynamic cryptographic verification shield (VERIFIED/TAMPERED)
│   │   │   └── ProofExporter.tsx   # 1-Click SOC 2 proof bundle generator
│   │   ├── App.tsx                 # Root compliance dashboard
│   │   ├── main.tsx                # React 19 entrypoint
│   │   └── index.css               # Tailwind CSS declarations
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── Makefile                        # Build, test, cross-compile, and proto generation
├── Dockerfile                      # Ultra-lightweight container image (< 30MB)
├── README.md                       # Comprehensive guide and SOC 2 onboarding manual
└── MASTER_PLAN.md                  # This master specification
```

---

## 5. Event Data Model & Cryptographic Verification Formula

### Standardized Audit Event Schema

```json
{
  "id": "evt_01JBCDEFGH890123456789",
  "timestamp": "2026-11-05T19:00:00.000Z",
  "tenant_id": "org_enterprise_acme",
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
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    "session_id": "sess_89f72b1a"
  }
}
```

### Merkle Block Sealing Formula
For any block of events $B_k = \{e_1, e_2, \dots, e_m\}$:
1. Each event is serialized to Canonical JSON (RFC 8785) and hashed:
   $$\text{Leaf}_i = \text{Blake3}(\text{CanonicalJSON}(e_i))$$
2. The Merkle root $R_k$ is computed over the leaves:
   $$R_k = \text{MerkleRoot}(\text{Leaf}_1, \dots, \text{Leaf}_m)$$
3. The block header is sealed with the previous block's hash:
   $$\text{BlockHash}_k = \text{Blake3}(R_k \parallel \text{BlockHash}_{k-1} \parallel \text{Timestamp}_k \parallel \text{Nonce})$$

If an attacker modifies a past event $e_x$ in block $B_j$ ($j < k$):
$$\text{Leaf}_x' \neq \text{Leaf}_x \implies R_j' \neq R_j \implies \text{BlockHash}_j' \neq \text{BlockHash}_j \implies \text{Broken Chain at Block } j$$

---

## 6. Implementation Roadmap & Milestones

### Phase 1: Cryptographic Ledger Core & Canonical Merkle Tree
- [x] Initialize Go module `github.com/alexandrmotologa/auditforge`.
- [x] Implement RFC 8785 canonical JSON sorting and Blake3/SHA-256 leaf hashing (`internal/ledger/canonical.go`, `internal/ledger/hash.go`).
- [x] Build binary Merkle Tree structure with proof path extraction and verification (`internal/ledger/merkle.go`).
- [x] Build cryptographic block models and hash-chain linking (`internal/ledger/block.go`).
- [x] Build append-only Write-Ahead Log (WAL) with binary framing (Magic bytes, length, CRC32, payload, boundary marker) and crash recovery (`internal/ledger/wal.go`).
- [x] Implement lock-free in-memory batch ring buffer accumulator (`internal/ledger/accumulator.go`).
- [x] Build continuous chain integrity verifier (`internal/ledger/verifier.go`).
- [x] Comprehensive unit tests: verify canonicalization, Merkle proofs, WAL durability, and tamper detection on deliberate bit-flips.

### Phase 2: Embedded Secondary Search Index (SQLite)
- [x] Implement pure-Go embedded SQLite schema and connection manager (`internal/index/sqlite.go`).
- [x] Implement indexed event search, multi-field filtering, pagination, and block lookups (`internal/index/query.go`).
- [x] Implement WAL-to-SQLite rebuild/re-index engine (`internal/index/reindex.go`).

### Phase 3: Ingestion APIs (REST, gRPC, SSE) & Security
- [x] Protocol Buffers schema definition (`internal/api/grpc/proto/audit.proto`).
- [x] Generate Go protobuf and gRPC server stubs.
- [x] Implement gRPC ingestion server (`internal/api/grpc/server.go`).
- [x] Implement HTTP REST API with routing, API key authentication, and rate limiting (`internal/api/rest/handler.go`).
- [x] Implement Server-Sent Events (SSE) live audit stream (`internal/api/rest/sse.go`).
- [x] Prometheus metrics endpoint (`/metrics`) for ingestion throughput, block sealing latency, and event counts.

### Phase 4: Proof Packaging, Ed25519 Signing & Standalone Offline CLI Verifier
- [x] Implement Ed25519 compliance key generation and block signing (`internal/ledger/signer.go`).
- [x] Implement `.proof` bundle archive generator (`internal/export/proof_bundle.go`).
- [x] Implement printable HTML/PDF SOC 2 compliance summary report generator (`internal/export/report.go`).
- [x] Build standalone `cmd/auditverify` CLI tool for offline verification with colored terminal outputs.
- [x] Build `cmd/auditforge` daemon CLI with subcommands (`serve`, `verify`, `export-proof`, `reindex`, `simulate-tamper`).

### Phase 5: Embedded React 19 Compliance Cockpit
- [x] Scaffold React 19 + TypeScript + Vite + Tailwind CSS frontend in `web/`.
- [x] Implement Live Integrity Shield (real-time VERIFIED / TAMPERED indicator).
- [x] Implement Live Event Stream with SSE auto-refresh and pause/resume.
- [x] Implement Filter & Search cockpit (tenants, actors, actions, targets, date ranges).
- [x] Implement Event Detail & Cryptographic Merkle Inspector Modal (visual leaf-to-root audit path).
- [x] Implement 1-Click SOC 2 / HIPAA `.proof` export dialog and compliance summary report viewer.
- [x] Implement Interactive Tamper Simulation tester tab to demonstrate instantaneous tamper detection.
- [x] Embed static assets directly into the Go binary using `//go:embed`.

### Phase 6: Automated End-to-End Testing, Benchmarking & Packaging
- [x] Write end-to-end integration tests (ingestion -> sealing -> querying -> proof export -> offline verification).
- [x] Write tamper detection integration test verifying that any bit flip fails offline verification.
- [x] Run benchmarks for ingestion throughput and Merkle tree generation.
- [x] Build multi-stage lightweight Dockerfile and Makefile.

### Phase 7: Advanced Enterprise Features & Client SDKs
- [x] Role-Based Access Control (RBAC) API Key store with role enforcement (`ingest_only`, `auditor_read`, `admin`) (`internal/auth/`).
- [x] External RFC 3161 Timestamping and notary witnessing (`internal/notary/`).
- [x] Real-time SIEM webhook log forwarder with HMAC-SHA256 request signatures (`internal/forwarder/`).
- [x] Automated tamper alerting dispatcher (Slack, PagerDuty, Webhooks) (`internal/alerting/`).
- [x] Native Go client SDK with asynchronous batching buffer (`sdk/go/`).
- [x] Native TypeScript / Node.js client SDK with zero dependencies (`sdk/ts/`).
- [x] React 19 Visual State-Change Diff Viewer (`web/src/components/EventDetailModal.tsx`).
- [x] Dedicated Merkle DAG Block Chain Explorer Tab (`web/src/components/BlockChainExplorer.tsx`).
- [x] One-Click SOC 2 / HIPAA Quick Filter Categories (`web/src/components/AuditFeed.tsx`).
- [x] Append-only WAL segment rotation and cold-storage archiving CLI (`internal/ledger/wal_rotator.go`).

