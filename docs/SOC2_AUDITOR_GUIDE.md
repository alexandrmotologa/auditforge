# SOC 2 & HIPAA Auditor Technical Guide

This guide explains the cryptographic guarantees, verification procedures, and audit trail structure of `auditforge`.

## Cryptographic Guarantees

### 1. Tamper Evidence
Every audit event ingested into `auditforge` is canonicalized using RFC 8785 JSON rules to ensure byte-level determinism regardless of key order, number formatting, or whitespace.

Each canonicalized event forms a leaf in a binary Merkle Tree:
$$\text{Leaf}_i = \text{Blake3}(\text{CanonicalJSON}(e_i))$$

Leaves are paired sequentially up to a single 32-byte Merkle root $R_k$.

### 2. Hash Chaining
Blocks are committed to an append-only Write-Ahead Log. Each block header contains:
* `block_index`: Monotonically increasing sequence number (uint64).
* `merkle_root`: Merkle root of the events in the current block.
* `previous_block_hash`: Cryptographic hash of block header $k-1$.
* `timestamp`: Sealed UTC timestamp (RFC 3339 nano).
* `event_count`: Number of events sealed in the block.
* `block_hash`: $\text{Blake3}(\text{block\_index} \parallel \text{merkle\_root} \parallel \text{previous\_block\_hash} \parallel \text{timestamp} \parallel \text{event\_count})$.

Any retroactive modification to an event alters its leaf hash, which invalidates the Merkle root, thereby invalidating the block hash and breaking every subsequent link in the hash chain.

### 3. Non-Repudiation and Offline Verification
Auditors do not need direct access to production databases or server infrastructure. A compliance officer can generate a `.proof` package for any time range. The auditor validates the bundle locally with `auditverify`:

```bash
auditverify verify compliance_q3_2026.proof
```

The verifier executes the following checks:
1. Recalculates the leaf hash for every event in the proof bundle.
2. Reconstructs the Merkle Tree for every block and verifies that the recalculated root matches the block header.
3. Validates the hash chain continuity ($H_{k-1}$ matches the previous hash pointer in block $k$).
4. Confirms that timestamps are non-decreasing.

## Data Retention and Immutability
* `auditforge` writes blocks sequentially to `.wal` files.
* SQLite is used exclusively as a read index for search performance (actor, action, target, date range). If the SQLite database is deleted or corrupted, it can be re-indexed directly from the immutable `.wal` files.
