package index

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
	_ "modernc.org/sqlite"
)

// Indexer manages the embedded SQLite secondary search database
type Indexer struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewIndexer opens or creates the SQLite index database and ensures schemas are migrated
func NewIndexer(dbPath string) (*Indexer, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed creating index db directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed opening sqlite index: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	indexer := &Indexer{db: db}
	if err := indexer.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("index schema migration failed: %w", err)
	}

	return indexer, nil
}

func (idx *Indexer) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS blocks (
		block_index INTEGER PRIMARY KEY,
		block_hash TEXT NOT NULL,
		previous_hash TEXT NOT NULL,
		merkle_root TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		event_count INTEGER NOT NULL,
		algorithm TEXT NOT NULL,
		signature TEXT,
		public_key TEXT,
		key_id TEXT,
		wal_offset INTEGER NOT NULL DEFAULT 0,
		raw_json TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		block_index INTEGER NOT NULL,
		leaf_index INTEGER NOT NULL,
		timestamp TEXT NOT NULL,
		tenant_id TEXT NOT NULL,
		actor_id TEXT NOT NULL,
		actor_name TEXT,
		actor_email TEXT,
		actor_role TEXT,
		action TEXT NOT NULL,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		target_name TEXT,
		leaf_hash TEXT NOT NULL,
		raw_json TEXT NOT NULL,
		FOREIGN KEY (block_index) REFERENCES blocks(block_index)
	);

	CREATE INDEX IF NOT EXISTS idx_events_tenant_time ON events (tenant_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_events_actor ON events (actor_id);
	CREATE INDEX IF NOT EXISTS idx_events_action ON events (action);
	CREATE INDEX IF NOT EXISTS idx_events_target ON events (target_type, target_id);
	CREATE INDEX IF NOT EXISTS idx_events_block ON events (block_index);
	`

	_, err := idx.db.Exec(schema)
	return err
}

// IndexBlock atomically inserts a sealed block and all of its encapsulated events into SQLite
func (idx *Indexer) IndexBlock(ctx context.Context, block *ledger.Block, walOffset int64) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed starting index transaction: %w", err)
	}
	defer tx.Rollback()

	blockRaw, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed marshaling block %d for index: %w", block.Header.Index, err)
	}

	blockStmt := `
	INSERT OR REPLACE INTO blocks (
		block_index, block_hash, previous_hash, merkle_root, timestamp,
		event_count, algorithm, signature, public_key, key_id, wal_offset, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`

	var sigHex, pubHex string
	if len(block.Header.Signature) > 0 {
		sigHex = hex.EncodeToString(block.Header.Signature)
	}
	if len(block.Header.PublicKey) > 0 {
		pubHex = hex.EncodeToString(block.Header.PublicKey)
	}

	_, err = tx.ExecContext(ctx, blockStmt,
		block.Header.Index,
		block.Header.BlockHashHex,
		block.Header.PreviousHex,
		block.Header.RootHex,
		block.Header.Timestamp.UTC().Format(time.RFC3339Nano),
		block.Header.EventCount,
		block.Header.Algorithm,
		sigHex,
		pubHex,
		block.Header.KeyID,
		walOffset,
		string(blockRaw),
	)
	if err != nil {
		return fmt.Errorf("failed inserting block %d into index: %w", block.Header.Index, err)
	}

	eventStmt, err := tx.PrepareContext(ctx, `
	INSERT OR REPLACE INTO events (
		id, block_index, leaf_index, timestamp, tenant_id,
		actor_id, actor_name, actor_email, actor_role,
		action, target_type, target_id, target_name, leaf_hash, raw_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`)
	if err != nil {
		return fmt.Errorf("failed preparing event insert statement: %w", err)
	}
	defer eventStmt.Close()

	for _, evt := range block.Events {
		rawEvt, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("failed marshaling event %s: %w", evt.ID, err)
		}

		_, err = eventStmt.ExecContext(ctx,
			evt.ID,
			evt.BlockIndex,
			evt.LeafIndex,
			evt.Timestamp.UTC().Format(time.RFC3339Nano),
			evt.TenantID,
			evt.Actor.ID,
			evt.Actor.Name,
			evt.Actor.Email,
			evt.Actor.Role,
			evt.Action,
			evt.Target.Type,
			evt.Target.ID,
			evt.Target.Name,
			evt.LeafHash,
			string(rawEvt),
		)
		if err != nil {
			return fmt.Errorf("failed indexing event %s: %w", evt.ID, err)
		}
	}

	return tx.Commit()
}

// Close closes the underlying SQLite database connection
func (idx *Indexer) Close() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if idx.db != nil {
		return idx.db.Close()
	}
	return nil
}
