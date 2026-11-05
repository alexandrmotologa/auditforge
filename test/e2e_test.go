package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	grpc_server "github.com/alexandrmotologa/auditforge/internal/api/grpc"
	pb "github.com/alexandrmotologa/auditforge/internal/api/grpc/proto"
	"github.com/alexandrmotologa/auditforge/internal/api/rest"
	"github.com/alexandrmotologa/auditforge/internal/export"
	"github.com/alexandrmotologa/auditforge/internal/index"
	"github.com/alexandrmotologa/auditforge/internal/ledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestEndToEnd_FullComplianceLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "e2e_ledger.wal")
	dbPath := filepath.Join(tempDir, "e2e_index.db")
	proofPath := filepath.Join(tempDir, "e2e_export.proof")

	// 1. Initialize Core Subsystems
	wal, err := ledger.OpenWAL(walPath)
	if err != nil {
		t.Fatalf("failed opening wal: %v", err)
	}
	defer wal.Close()

	indexer, err := index.NewIndexer(dbPath)
	if err != nil {
		t.Fatalf("failed opening indexer: %v", err)
	}
	defer indexer.Close()

	signer, err := ledger.NewEd25519Signer()
	if err != nil {
		t.Fatalf("failed creating signer: %v", err)
	}

	sseHub := rest.NewSSEHub()
	defer sseHub.Close()

	accCfg := ledger.AccumulatorConfig{
		BatchSize:     5,
		FlushInterval: 50 * time.Millisecond,
		Algorithm:     ledger.AlgorithmBlake3,
	}

	acc, err := ledger.NewAccumulator(accCfg, wal, signer, func(block *ledger.Block, tree *ledger.MerkleTree) error {
		_ = indexer.IndexBlock(context.Background(), block, 0)
		sseHub.BroadcastEvent("block_sealed", block.Header)
		return nil
	})
	if err != nil {
		t.Fatalf("failed creating accumulator: %v", err)
	}
	defer acc.Close()

	// 2. Start REST & gRPC Servers
	restHandler := rest.NewHandler(acc, indexer, wal, sseHub, ledger.AlgorithmBlake3, func(k string) bool { return true })
	httpServer := httptest.NewServer(restHandler)
	defer httpServer.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterAuditServiceServer(grpcServer, grpc_server.NewServer(acc, indexer, ledger.AlgorithmBlake3))
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed listening on random grpc port: %v", err)
	}
	go func() { _ = grpcServer.Serve(lis) }()
	defer grpcServer.Stop()

	// 3. Ingest 10 events via REST
	client := &http.Client{Timeout: 5 * time.Second}
	var restEventIDs []string

	for i := 0; i < 10; i++ {
		payload := map[string]any{
			"tenant_id": "org_acme",
			"actor": map[string]any{
				"id":    fmt.Sprintf("usr_%d", i),
				"name":  fmt.Sprintf("User %d", i),
				"email": fmt.Sprintf("user%d@acme.com", i),
				"role":  "engineer",
			},
			"action": "repo.push",
			"target": map[string]any{
				"id":   "repo_auditforge",
				"type": "repository",
			},
		}

		b, _ := json.Marshal(payload)
		resp, err := client.Post(httpServer.URL+"/api/v1/events?sync=true", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("rest ingest failed: %v", err)
		}
		var res map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		restEventIDs = append(restEventIDs, res["event_id"].(string))
	}

	// 4. Ingest 10 events via gRPC
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed grpc client connection: %v", err)
	}
	defer conn.Close()
	grpcClient := pb.NewAuditServiceClient(conn)

	for i := 0; i < 10; i++ {
		gResp, err := grpcClient.IngestEvent(context.Background(), &pb.AuditEventRequest{
			TenantId: "org_enterprise",
			Action:   "policy.evaluate",
			Actor: &pb.ActorMsg{
				Id:   fmt.Sprintf("svc_%d", i),
				Name: "Compliance Agent",
			},
			Target: &pb.TargetMsg{
				Id:   "pol_soc2",
				Type: "compliance_policy",
			},
			Sync: true,
		})
		if err != nil {
			t.Fatalf("grpc ingest failed: %v", err)
		}
		if gResp.Status != "sealed" {
			t.Fatalf("expected status sealed, got %s", gResp.Status)
		}
	}

	// 5. Query Search Index with Filters
	searchResp, err := client.Get(httpServer.URL + "/api/v1/events?tenant_id=org_acme")
	if err != nil {
		t.Fatalf("search request failed: %v", err)
	}
	var searchRes map[string]any
	_ = json.NewDecoder(searchResp.Body).Decode(&searchRes)
	searchResp.Body.Close()

	if int(searchRes["total"].(float64)) != 10 {
		t.Fatalf("expected 10 events for org_acme, got %v", searchRes["total"])
	}

	// 6. Inspect Event Proof & Merkle Verification
	inspectResp, err := client.Get(httpServer.URL + "/api/v1/events/" + restEventIDs[3])
	if err != nil {
		t.Fatalf("inspect request failed: %v", err)
	}
	var inspectRes map[string]any
	_ = json.NewDecoder(inspectResp.Body).Decode(&inspectRes)
	inspectResp.Body.Close()

	if inspectRes["verified"] != true {
		t.Fatalf("expected inspected event to be verified, got: %v", inspectRes["verified"])
	}

	// 7. Export Proof Bundle for Auditor
	manifest, err := export.GenerateProofBundle(wal, 0, 999999, signer, proofPath)
	if err != nil {
		t.Fatalf("failed generating proof bundle: %v", err)
	}

	if manifest.TotalEvents != 20 {
		t.Fatalf("expected 20 certified events in manifest, got %d", manifest.TotalEvents)
	}

	// 8. Offline Verification (Read .proof package)
	readManifest, chain, events, err := export.ReadProofBundle(proofPath)
	if err != nil {
		t.Fatalf("offline read failed: %v", err)
	}

	if len(events) != 20 || len(chain) != int(manifest.TotalBlocks) {
		t.Fatalf("offline package mismatch: %d events, %d blocks", len(events), len(chain))
	}

	// Verify all 20 events offline
	for _, pe := range events {
		calcHash, err := pe.Event.ComputeLeafHash(readManifest.Algorithm)
		if err != nil {
			t.Fatalf("offline leaf hash calculation failed: %v", err)
		}
		if !ledger.VerifyHash(calcHash, pe.Event.LeafHash) {
			t.Fatalf("offline leaf hash mismatch for event %s", pe.Event.ID)
		}
		if !ledger.VerifyMerkleProof(pe.Proof) {
			t.Fatalf("offline merkle proof verification failed for event %s", pe.Event.ID)
		}
	}

	// 9. Verify Chain Integrity via Verify API
	vResp, err := client.Get(httpServer.URL + "/api/v1/verify")
	if err != nil {
		t.Fatalf("verify request failed: %v", err)
	}
	var vRes ledger.VerificationResult
	_ = json.NewDecoder(vResp.Body).Decode(&vRes)
	vResp.Body.Close()

	if !vRes.Valid || vRes.TotalEvents != 20 {
		t.Fatalf("ledger verification expected valid with 20 events, got: %+v", vRes)
	}
}
