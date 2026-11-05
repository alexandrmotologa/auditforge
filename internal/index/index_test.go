package index

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

func TestIndexer_Operations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_index.db")

	idx, err := NewIndexer(dbPath)
	if err != nil {
		t.Fatalf("failed initializing indexer: %v", err)
	}
	defer idx.Close()

	ctx := context.Background()

	// Create test events and block
	now := time.Now().UTC()
	events := []*ledger.AuditEvent{
		{
			ID:        "evt_001",
			Timestamp: now.Add(-2 * time.Minute),
			TenantID:  "org_acme",
			Actor: ledger.Actor{
				ID:    "usr_alex",
				Name:  "Alexandr Motologa",
				Email: "alex@example.com",
				Role:  "security_engineer",
			},
			Action: "credential.create",
			Target: ledger.Target{
				ID:   "cred_prod_api",
				Type: "api_key",
				Name: "Production Ingestion Key",
			},
			Metadata: map[string]any{"ip": "10.0.0.1"},
		},
		{
			ID:        "evt_002",
			Timestamp: now.Add(-1 * time.Minute),
			TenantID:  "org_acme",
			Actor: ledger.Actor{
				ID:    "usr_bob",
				Name:  "Bob Auditor",
				Email: "bob@auditor.com",
				Role:  "auditor",
			},
			Action: "data.export",
			Target: ledger.Target{
				ID:   "exp_audit_q3",
				Type: "proof_bundle",
				Name: "Q3 Compliance Bundle",
			},
			Metadata: map[string]any{"ip": "10.0.0.2"},
		},
		{
			ID:        "evt_003",
			Timestamp: now,
			TenantID:  "org_other",
			Actor: ledger.Actor{
				ID:   "usr_carol",
				Name: "Carol Dev",
			},
			Action: "service.deploy",
			Target: ledger.Target{
				ID:   "svc_auth",
				Type: "service",
			},
		},
	}

	block, _, err := ledger.NewBlock(0, make([]byte, 32), events, ledger.AlgorithmBlake3)
	if err != nil {
		t.Fatalf("failed creating test block: %v", err)
	}
	_ = block.Seal(nil)

	if err := idx.IndexBlock(ctx, block, 0); err != nil {
		t.Fatalf("failed indexing block: %v", err)
	}

	// 1. Query by tenant
	res, err := idx.SearchEvents(ctx, EventFilter{TenantID: "org_acme"})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if res.Total != 2 {
		t.Errorf("expected 2 events for org_acme, got %d", res.Total)
	}

	// 2. Query by action
	resAction, err := idx.SearchEvents(ctx, EventFilter{Action: "data.export"})
	if err != nil {
		t.Fatalf("search by action failed: %v", err)
	}
	if resAction.Total != 1 || resAction.Events[0].ID != "evt_002" {
		t.Errorf("expected evt_002 for data.export, got %v", resAction.Events)
	}

	// 3. Fuzzy search query
	resQuery, err := idx.SearchEvents(ctx, EventFilter{Query: "Motologa"})
	if err != nil {
		t.Fatalf("fuzzy search failed: %v", err)
	}
	if resQuery.Total != 1 || resQuery.Events[0].ID != "evt_001" {
		t.Errorf("expected 1 match for query Motologa, got %d", resQuery.Total)
	}

	// 4. Get by ID
	evt, err := idx.GetEventByID(ctx, "evt_001")
	if err != nil || evt == nil {
		t.Fatalf("failed retrieving event by id: %v", err)
	}
	if evt.Actor.Name != "Alexandr Motologa" {
		t.Errorf("retrieved actor name mismatch: %s", evt.Actor.Name)
	}

	// 5. Ledger Stats
	stats, err := idx.GetLedgerStats(ctx)
	if err != nil {
		t.Fatalf("failed getting stats: %v", err)
	}
	if stats.TotalEvents != 3 || stats.TotalBlocks != 1 || stats.TenantCount != 2 {
		t.Errorf("stats mismatch: %+v", stats)
	}
}

func TestIndexer_ReindexFromWAL(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "reindex.wal")
	dbPath := filepath.Join(tempDir, "reindex.db")

	wal, _ := ledger.OpenWAL(walPath)
	defer wal.Close()

	evts := []*ledger.AuditEvent{
		{
			ID:        "evt_wal_1",
			Timestamp: time.Now().UTC(),
			TenantID:  "org_wal",
			Actor:     ledger.Actor{ID: "usr_1"},
			Action:    "login",
			Target:    ledger.Target{ID: "sys"},
		},
	}
	b, _, _ := ledger.NewBlock(0, make([]byte, 32), evts, ledger.AlgorithmBlake3)
	_ = b.Seal(nil)
	_, _ = wal.AppendBlock(b)

	idx, _ := NewIndexer(dbPath)
	defer idx.Close()

	ctx := context.Background()
	blocks, events, err := ReindexFromWAL(ctx, wal, idx)
	if err != nil {
		t.Fatalf("reindex from wal failed: %v", err)
	}
	if blocks != 1 || events != 1 {
		t.Fatalf("expected 1 block and 1 event, got %d and %d", blocks, events)
	}

	retrieved, err := idx.GetEventByID(ctx, "evt_wal_1")
	if err != nil || retrieved == nil {
		t.Fatalf("failed finding reindexed event: %v", err)
	}
}
