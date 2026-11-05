export interface Actor {
  id: string;
  name?: string;
  email?: string;
  role?: string;
  metadata?: Record<string, any>;
}

export interface Target {
  id: string;
  type: string;
  name?: string;
  metadata?: Record<string, any>;
}

export interface AuditEvent {
  id: string;
  timestamp: string;
  tenant_id: string;
  actor: Actor;
  action: string;
  target: Target;
  metadata?: Record<string, any>;
  leaf_hash: string;
  block_index: number;
  leaf_index: number;
}

export interface BlockHeader {
  index: number;
  previous_hash: string;
  previous_hex: string;
  merkle_root: string;
  root_hex: string;
  timestamp: string;
  event_count: number;
  algorithm: string;
  signature?: string;
  public_key?: string;
  key_id?: string;
  block_hash: string;
  block_hash_hex: string;
}

export interface MerkleProofStep {
  position: 'left' | 'right';
  sibling_hex: string;
}

export interface MerkleProof {
  leaf_index: number;
  leaf_hex: string;
  audit_path: MerkleProofStep[];
  root_hex: string;
  algorithm: string;
}

export interface VerificationResult {
  valid: boolean;
  total_blocks: number;
  total_events: number;
  tampered_block_index?: number;
  reason?: string;
  duration_ms: number;
  head_block_hash?: string;
  genesis_block_hash?: string;
  verified_at: string;
}

export interface LedgerStats {
  total_events: number;
  total_blocks: number;
  tenant_count: number;
  first_event_time?: string;
  last_event_time?: string;
  ledger_verified: boolean;
  head_block_hash?: string;
  algorithm: string;
  tamper_simulation: boolean;
}
