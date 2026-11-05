package index

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// ReindexFromWAL clears the SQLite search database and rebuilds all records directly from the WAL
func ReindexFromWAL(ctx context.Context, wal *ledger.WAL, indexer *Indexer) (blocksIndexed int, eventsIndexed int, err error) {
	// Clear existing secondary data
	_, err = indexer.db.ExecContext(ctx, "DELETE FROM events; DELETE FROM blocks;")
	if err != nil {
		return 0, 0, fmt.Errorf("failed clearing index tables: %w", err)
	}

	blocks, err := wal.ReadAllBlocks()
	if err != nil {
		return 0, 0, fmt.Errorf("failed reading blocks from wal: %w", err)
	}

	for _, block := range blocks {
		// Index block (offset can be set to 0 or recovered)
		if err := indexer.IndexBlock(ctx, block, 0); err != nil {
			return blocksIndexed, eventsIndexed, fmt.Errorf("failed re-indexing block %d: %w", block.Header.Index, err)
		}
		blocksIndexed++
		eventsIndexed += len(block.Events)
	}

	return blocksIndexed, eventsIndexed, nil
}
