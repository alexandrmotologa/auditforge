package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// EventFilter specifies search criteria for audit event queries
type EventFilter struct {
	TenantID   string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Query      string
	BlockIndex *uint64
	Since      *time.Time
	Until      *time.Time
	Limit      int
	Offset     int
}

// EventQueryResult packages paginated search results
type EventQueryResult struct {
	Events []*ledger.AuditEvent `json:"events"`
	Total  int64                `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

// BlockQueryResult packages paginated block listing results
type BlockQueryResult struct {
	Blocks []*ledger.BlockHeader `json:"blocks"`
	Total  int64                 `json:"total"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
}

// LedgerStats reports summary metrics from the search index
type LedgerStats struct {
	TotalEvents    int64      `json:"total_events"`
	TotalBlocks    int64      `json:"total_blocks"`
	TenantCount    int64      `json:"tenant_count"`
	FirstEventTime *time.Time `json:"first_event_time,omitempty"`
	LastEventTime  *time.Time `json:"last_event_time,omitempty"`
}

// SearchEvents executes a multi-field indexed query with pagination
func (idx *Indexer) SearchEvents(ctx context.Context, filter EventFilter) (*EventQueryResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var whereClauses []string
	var args []any

	if filter.TenantID != "" {
		whereClauses = append(whereClauses, "tenant_id = ?")
		args = append(args, filter.TenantID)
	}
	if filter.ActorID != "" {
		whereClauses = append(whereClauses, "actor_id = ?")
		args = append(args, filter.ActorID)
	}
	if filter.Action != "" {
		whereClauses = append(whereClauses, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.TargetType != "" {
		whereClauses = append(whereClauses, "target_type = ?")
		args = append(args, filter.TargetType)
	}
	if filter.TargetID != "" {
		whereClauses = append(whereClauses, "target_id = ?")
		args = append(args, filter.TargetID)
	}
	if filter.BlockIndex != nil {
		whereClauses = append(whereClauses, "block_index = ?")
		args = append(args, *filter.BlockIndex)
	}
	if filter.Since != nil {
		whereClauses = append(whereClauses, "timestamp >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339Nano))
	}
	if filter.Until != nil {
		whereClauses = append(whereClauses, "timestamp <= ?")
		args = append(args, filter.Until.UTC().Format(time.RFC3339Nano))
	}
	if filter.Query != "" {
		q := "%" + strings.ToLower(filter.Query) + "%"
		whereClauses = append(whereClauses, "(LOWER(action) LIKE ? OR LOWER(actor_name) LIKE ? OR LOWER(target_name) LIKE ? OR LOWER(id) LIKE ?)")
		args = append(args, q, q, q, q)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total matches
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events %s;", whereSQL)
	var total int64
	if err := idx.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed counting search events: %w", err)
	}

	// Retrieve paginated rows
	selectSQL := fmt.Sprintf(`
	SELECT raw_json FROM events
	%s
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?;
	`, whereSQL)

	queryArgs := append(args, filter.Limit, filter.Offset)
	rows, err := idx.db.QueryContext(ctx, selectSQL, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed querying events: %w", err)
	}
	defer rows.Close()

	events := make([]*ledger.AuditEvent, 0, filter.Limit)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("failed scanning event row: %w", err)
		}
		var evt ledger.AuditEvent
		if err := json.Unmarshal([]byte(raw), &evt); err != nil {
			return nil, fmt.Errorf("failed unmarshaling indexed event: %w", err)
		}
		events = append(events, &evt)
	}

	return &EventQueryResult{
		Events: events,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

// GetEventByID retrieves a single event by its unique ID
func (idx *Indexer) GetEventByID(ctx context.Context, id string) (*ledger.AuditEvent, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var raw string
	err := idx.db.QueryRowContext(ctx, "SELECT raw_json FROM events WHERE id = ?;", id).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error querying event %s: %w", id, err)
	}

	var evt ledger.AuditEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		return nil, fmt.Errorf("error unmarshaling event %s: %w", id, err)
	}

	return &evt, nil
}

// GetBlockByIndex returns the complete block (including events) by block index
func (idx *Indexer) GetBlockByIndex(ctx context.Context, blockIndex uint64) (*ledger.Block, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var raw string
	err := idx.db.QueryRowContext(ctx, "SELECT raw_json FROM blocks WHERE block_index = ?;", blockIndex).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error querying block %d: %w", blockIndex, err)
	}

	var b ledger.Block
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		return nil, fmt.Errorf("error unmarshaling block %d: %w", blockIndex, err)
	}

	return &b, nil
}

// ListBlocks returns paginated block headers in reverse chronological order
func (idx *Indexer) ListBlocks(ctx context.Context, limit, offset int) (*BlockQueryResult, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := idx.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blocks;").Scan(&total); err != nil {
		return nil, fmt.Errorf("failed counting blocks: %w", err)
	}

	rows, err := idx.db.QueryContext(ctx, `
	SELECT raw_json FROM blocks
	ORDER BY block_index DESC
	LIMIT ? OFFSET ?;
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed querying block headers: %w", err)
	}
	defer rows.Close()

	headers := make([]*ledger.BlockHeader, 0, limit)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("failed scanning block row: %w", err)
		}
		var b ledger.Block
		if err := json.Unmarshal([]byte(raw), &b); err != nil {
			return nil, fmt.Errorf("failed unmarshaling block header: %w", err)
		}
		headers = append(headers, &b.Header)
	}

	return &BlockQueryResult{
		Blocks: headers,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetLedgerStats gathers high-level index metrics
func (idx *Indexer) GetLedgerStats(ctx context.Context) (*LedgerStats, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	stats := &LedgerStats{}

	_ = idx.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events;").Scan(&stats.TotalEvents)
	_ = idx.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM blocks;").Scan(&stats.TotalBlocks)
	_ = idx.db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT tenant_id) FROM events;").Scan(&stats.TenantCount)

	var firstTS, lastTS sql.NullString
	_ = idx.db.QueryRowContext(ctx, "SELECT MIN(timestamp), MAX(timestamp) FROM events;").Scan(&firstTS, &lastTS)

	if firstTS.Valid && firstTS.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, firstTS.String); err == nil {
			stats.FirstEventTime = &t
		}
	}
	if lastTS.Valid && lastTS.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastTS.String); err == nil {
			stats.LastEventTime = &t
		}
	}

	return stats, nil
}
