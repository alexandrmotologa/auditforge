package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	grpc_server "github.com/alexandrmotologa/auditforge/internal/api/grpc"
	pb "github.com/alexandrmotologa/auditforge/internal/api/grpc/proto"
	"github.com/alexandrmotologa/auditforge/internal/api/rest"
	"github.com/alexandrmotologa/auditforge/internal/auth"
	"github.com/alexandrmotologa/auditforge/internal/index"
	"github.com/alexandrmotologa/auditforge/internal/ledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func setupTestEnvironment(t *testing.T) (*ledger.Accumulator, *index.Indexer, *ledger.WAL, *rest.SSEHub, func()) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")
	dbPath := filepath.Join(tempDir, "test.db")

	wal, err := ledger.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}

	idx, err := index.NewIndexer(dbPath)
	if err != nil {
		t.Fatalf("failed opening indexer: %v", err)
	}

	sseHub := rest.NewSSEHub()
	signer, _ := ledger.NewEd25519Signer()

	cfg := ledger.DefaultAccumulatorConfig()
	cfg.FlushInterval = 50 * time.Millisecond

	acc, err := ledger.NewAccumulator(cfg, wal, signer, func(block *ledger.Block, tree *ledger.MerkleTree) error {
		_ = idx.IndexBlock(context.Background(), block, 0)
		sseHub.BroadcastEvent("block_sealed", block.Header)
		return nil
	})
	if err != nil {
		t.Fatalf("failed initializing accumulator: %v", err)
	}

	cleanup := func() {
		acc.Close()
		wal.Close()
		idx.Close()
		sseHub.Close()
	}

	return acc, idx, wal, sseHub, cleanup
}

func TestREST_IngestAndQuery(t *testing.T) {
	acc, idx, wal, sseHub, cleanup := setupTestEnvironment(t)
	defer cleanup()

	keyStore, _ := auth.NewKeyStore("", []string{"test-valid-key"})
	handler := rest.NewHandler(acc, idx, wal, sseHub, keyStore, nil, nil, nil, ledger.AlgorithmBlake3)
	server := httptest.NewServer(handler)
	defer server.Close()

	// 1. Ingest event via REST with authentication
	evtPayload := map[string]any{
		"tenant_id": "org_compliance",
		"actor": map[string]any{
			"id":   "usr_alex",
			"name": "Alexandr Motologa",
		},
		"action": "permission.grant",
		"target": map[string]any{
			"id":   "res_vault",
			"type": "database",
		},
	}

	rawBody, _ := json.Marshal(evtPayload)
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/events?sync=true", bytes.NewReader(rawBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-valid-key")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("ingest request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", resp.StatusCode)
	}

	var ingestRes map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ingestRes)
	eventID := ingestRes["event_id"].(string)

	// 2. Query event with Merkle proof
	getReq, _ := http.NewRequest("GET", server.URL+"/api/v1/events/"+eventID, nil)
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}

	var detailRes map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&detailRes)
	if detailRes["verified"] != true {
		t.Fatalf("expected event proof verified to be true")
	}

	// 3. Verify entire ledger
	verifyReq, _ := http.NewRequest("GET", server.URL+"/api/v1/verify", nil)
	verifyResp, err := client.Do(verifyReq)
	if err != nil {
		t.Fatalf("verify request failed: %v", err)
	}
	defer verifyResp.Body.Close()

	var verifyRes ledger.VerificationResult
	_ = json.NewDecoder(verifyResp.Body).Decode(&verifyRes)
	if !verifyRes.Valid || verifyRes.TotalEvents != 1 {
		t.Fatalf("expected valid ledger with 1 event, got %+v", verifyRes)
	}
}

func TestGRPC_IngestAndVerify(t *testing.T) {
	acc, idx, _, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	grpcServer := grpc.NewServer()
	srv := grpc_server.NewServer(acc, idx, ledger.AlgorithmBlake3)
	pb.RegisterAuditServiceServer(grpcServer, srv)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on random port: %v", err)
	}
	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed connecting to grpc server: %v", err)
	}
	defer conn.Close()

	client := pb.NewAuditServiceClient(conn)
	ctx := context.Background()

	// Ingest synchronous event via gRPC
	res, err := client.IngestEvent(ctx, &pb.AuditEventRequest{
		TenantId: "org_grpc",
		Action:   "policy.update",
		Actor: &pb.ActorMsg{
			Id:   "usr_grpc",
			Name: "GRPC Ingester",
		},
		Target: &pb.TargetMsg{
			Id:   "pol_soc2",
			Type: "security_policy",
		},
		Sync: true,
	})
	if err != nil {
		t.Fatalf("grpc ingest failed: %v", err)
	}

	if res.Status != "sealed" || res.EventId == "" {
		t.Fatalf("unexpected grpc response: %+v", res)
	}

	// Verify event inclusion proof via gRPC
	vRes, err := client.VerifyEvent(ctx, &pb.VerifyEventRequest{EventId: res.EventId})
	if err != nil {
		t.Fatalf("grpc verify event failed: %v", err)
	}

	if !vRes.Valid || vRes.EventId != res.EventId {
		t.Fatalf("expected valid proof for event %s, got: %+v", res.EventId, vRes)
	}
}
