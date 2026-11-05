# auditforge Architecture Documentation

## Core Subsystems

`auditforge` consists of four main decoupled subsystems:

### 1. Ingestion Layer (`internal/api`)
* **REST HTTP Service**: Exposes JSON endpoints for single-event and batch-event ingestion (`POST /api/v1/events`, `POST /api/v1/events/batch`).
* **gRPC Service**: Defined in `internal/api/grpc/audit.proto` for low-latency, binary payload serialization in distributed enterprise services.
* **Server-Sent Events (SSE)**: Streams validated audit records to web clients in real time.

### 2. Cryptographic Ledger Engine (`internal/ledger`)
* **Canonical Serialization**: Implements RFC 8785 canonical JSON sorting so event signatures remain invariant across architectures.
* **Hashing Engine**: Uses Blake3 for primary high-performance hashing (with SHA-256 fallback compatibility).
* **Merkle Tree Generator**: Creates complete binary Merkle Trees and calculates audit paths (Merkle proofs) for individual events.
* **Block Sealer**: Accumulates ingested events into discrete cryptographic blocks and seals them with hash chain pointers.
* **Append-Only WAL**: Serializes sealed blocks into append-only binary files (`.wal`) with CRC32 frame checksums to prevent torn writes.

### 3. Read & Search Index (`internal/index`)
* Embeds SQLite (`modernc.org/sqlite` or pure Go SQLite) to maintain secondary index tables for fast multi-tenant filtering.
* Indexed fields: `tenant_id`, `actor_id`, `action`, `target_id`, `timestamp`, `block_id`.
* The index is completely derivative: it can be dropped and reconstructed from the raw `.wal` files at any time.

### 4. Verification & Export Subsystem (`internal/export`, `cmd/auditverify`)
* Packages event ranges, Merkle roots, and cryptographic block headers into `.proof` archives (compressed tarball containing structured JSON metadata).
* `auditverify` CLI parses `.proof` packages and verifies all cryptographic properties offline.

### 5. Web Compliance Cockpit (`web/`)
* React 19 single-page application built with Vite and Tailwind CSS.
* Provides real-time event inspection, search/filtering, tamper-detection status indicator, and one-click proof bundle downloads.
* Compiled and embedded into the Go executable via Go's `//go:embed` directive.
