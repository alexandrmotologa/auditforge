package grpc

import (
	"context"
	"time"

	pb "github.com/alexandrmotologa/auditforge/internal/api/grpc/proto"
	"github.com/alexandrmotologa/auditforge/internal/index"
	"github.com/alexandrmotologa/auditforge/internal/ledger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements the gRPC AuditService
type Server struct {
	pb.UnimplementedAuditServiceServer
	acc     *ledger.Accumulator
	indexer *index.Indexer
	algo    string
}

// NewServer constructs the gRPC service handler
func NewServer(acc *ledger.Accumulator, indexer *index.Indexer, algorithm string) *Server {
	return &Server{
		acc:     acc,
		indexer: indexer,
		algo:    ledger.NormalizeAlgorithm(algorithm),
	}
}

// IngestEvent processes a single audit event via gRPC
func (s *Server) IngestEvent(ctx context.Context, req *pb.AuditEventRequest) (*pb.AuditEventResponse, error) {
	if req.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "action is required")
	}

	evt := mapProtoToEvent(req)

	if req.Sync {
		block, leafHash, err := s.acc.IngestSync(ctx, evt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "sync ingestion failed: %v", err)
		}
		return &pb.AuditEventResponse{
			Status:      "sealed",
			EventId:     evt.ID,
			LeafHash:    leafHash,
			BlockIndex:  block.Header.Index,
			LeafIndex:   int32(evt.LeafIndex),
			ReceivedAt:  evt.Timestamp.UTC().Format(time.RFC3339Nano),
		}, nil
	}

	leafHash, err := s.acc.IngestAsync(evt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "async ingestion failed: %v", err)
	}

	return &pb.AuditEventResponse{
		Status:     "accepted",
		EventId:    evt.ID,
		LeafHash:   leafHash,
		ReceivedAt: evt.Timestamp.UTC().Format(time.RFC3339Nano),
	}, nil
}

// IngestBatch processes an array of audit events in a single gRPC call
func (s *Server) IngestBatch(ctx context.Context, req *pb.AuditBatchRequest) (*pb.AuditBatchResponse, error) {
	if len(req.Events) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one event required")
	}

	leafHashes := make([]string, 0, len(req.Events))
	for _, reqEvt := range req.Events {
		if reqEvt.TenantId == "" {
			reqEvt.TenantId = req.TenantId
		}
		evt := mapProtoToEvent(reqEvt)
		leafHash, err := s.acc.IngestAsync(evt)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed ingesting event %s: %v", evt.ID, err)
		}
		leafHashes = append(leafHashes, leafHash)
	}

	if req.Sync {
		if err := s.acc.Flush(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed flushing batch: %v", err)
		}
	}

	return &pb.AuditBatchResponse{
		Status:        "accepted",
		IngestedCount: int32(len(req.Events)),
		LeafHashes:    leafHashes,
	}, nil
}

// VerifyEvent retrieves an event, builds its Merkle audit path, and verifies inclusion against block root
func (s *Server) VerifyEvent(ctx context.Context, req *pb.VerifyEventRequest) (*pb.VerifyEventResponse, error) {
	if req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}

	evt, err := s.indexer.GetEventByID(ctx, req.EventId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed querying event: %v", err)
	}
	if evt == nil {
		return nil, status.Errorf(codes.NotFound, "event %s not found", req.EventId)
	}

	block, err := s.indexer.GetBlockByIndex(ctx, evt.BlockIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed querying block: %v", err)
	}
	if block == nil {
		return nil, status.Errorf(codes.NotFound, "block %d not found", evt.BlockIndex)
	}

	// Rebuild Merkle tree for this block to extract proof
	leaves := make([][]byte, len(block.Events))
	for i, e := range block.Events {
		lh, err := e.ComputeLeafHash(s.algo)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "leaf hash error: %v", err)
		}
		leaves[i] = lh
	}

	tree := ledger.NewMerkleTree(s.algo, leaves)
	proof, err := tree.GetProof(evt.LeafIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed computing proof: %v", err)
	}

	isValid := ledger.VerifyMerkleProof(proof)
	var steps []*pb.MerkleStepMsg
	for _, step := range proof.AuditPath {
		steps = append(steps, &pb.MerkleStepMsg{
			Position:   string(step.Position),
			SiblingHex: step.SiblingHex,
		})
	}

	return &pb.VerifyEventResponse{
		Valid:      isValid,
		EventId:    evt.ID,
		LeafHash:   evt.LeafHash,
		BlockIndex: evt.BlockIndex,
		MerkleRoot: block.Header.RootHex,
		ProofSteps: steps,
	}, nil
}

func mapProtoToEvent(req *pb.AuditEventRequest) *ledger.AuditEvent {
	id := req.Id
	if id == "" {
		id = ledger.NewEventID()
	}

	var ts time.Time
	if req.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, req.Timestamp); err == nil {
			ts = parsed.UTC()
		}
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	actorMeta := make(map[string]any)
	if req.Actor != nil {
		for k, v := range req.Actor.Metadata {
			actorMeta[k] = v
		}
	}

	targetMeta := make(map[string]any)
	if req.Target != nil {
		for k, v := range req.Target.Metadata {
			targetMeta[k] = v
		}
	}

	meta := make(map[string]any)
	for k, v := range req.Metadata {
		meta[k] = v
	}

	evt := &ledger.AuditEvent{
		ID:        id,
		Timestamp: ts,
		TenantID:  req.TenantId,
		Action:    req.Action,
		Metadata:  meta,
	}

	if req.Actor != nil {
		evt.Actor = ledger.Actor{
			ID:       req.Actor.Id,
			Name:     req.Actor.Name,
			Email:    req.Actor.Email,
			Role:     req.Actor.Role,
			Metadata: actorMeta,
		}
	}

	if req.Target != nil {
		evt.Target = ledger.Target{
			ID:       req.Target.Id,
			Type:     req.Target.Type,
			Name:     req.Target.Name,
			Metadata: targetMeta,
		}
	}

	return evt
}
