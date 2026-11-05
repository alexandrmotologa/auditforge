package rest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/auditforge/internal/index"
	"github.com/alexandrmotologa/auditforge/internal/ledger"
)

// APIKeyValidator validates incoming HTTP request credentials
type APIKeyValidator func(key string) bool

// Handler coordinates REST routing, authentication, and responses
type Handler struct {
	acc       *ledger.Accumulator
	indexer   *index.Indexer
	wal       *ledger.WAL
	sseHub    *SSEHub
	algo      string
	validator APIKeyValidator
	mux       *http.ServeMux
	simulatedTamperBlock *uint64
}

// NewHandler initializes all REST endpoints
func NewHandler(acc *ledger.Accumulator, indexer *index.Indexer, wal *ledger.WAL, sseHub *SSEHub, algo string, validator APIKeyValidator) *Handler {
	h := &Handler{
		acc:       acc,
		indexer:   indexer,
		wal:       wal,
		sseHub:    sseHub,
		algo:      ledger.NormalizeAlgorithm(algo),
		validator: validator,
		mux:       http.NewServeMux(),
	}

	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	// Ingestion
	h.mux.HandleFunc("POST /api/v1/events", h.authMiddleware(h.handleIngestEvent))
	h.mux.HandleFunc("POST /api/v1/events/batch", h.authMiddleware(h.handleIngestBatch))

	// Search & Query
	h.mux.HandleFunc("GET /api/v1/events", h.handleSearchEvents)
	h.mux.HandleFunc("GET /api/v1/events/", h.handleGetEventByID)
	h.mux.HandleFunc("GET /api/v1/blocks", h.handleListBlocks)
	h.mux.HandleFunc("GET /api/v1/blocks/", h.handleGetBlockByIndex)

	// Verification & Stats
	h.mux.HandleFunc("GET /api/v1/verify", h.handleVerifyLedger)
	h.mux.HandleFunc("GET /api/v1/stats", h.handleGetStats)
	h.mux.HandleFunc("GET /metrics", h.handleMetrics)

	// Real-time stream
	h.mux.HandleFunc("GET /api/v1/stream", h.sseHub.ServeHTTP)

	// Tamper simulation for interactive testing
	h.mux.HandleFunc("POST /api/v1/tamper-simulate", h.handleSimulateTamper)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.mux.ServeHTTP(w, r)
}

func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.validator == nil {
			next(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				key = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if !h.validator(key) {
			writeJSONError(w, http.StatusUnauthorized, "invalid or missing api key")
			return
		}

		next(w, r)
	}
}

func (h *Handler) handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	var evt ledger.AuditEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid json payload: %v", err))
		return
	}

	if evt.Action == "" {
		writeJSONError(w, http.StatusBadRequest, "action is required")
		return
	}

	isSync := r.URL.Query().Get("sync") == "true"
	if isSync {
		block, leafHash, err := h.acc.IngestSync(r.Context(), &evt)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("sync ingestion error: %v", err))
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"status":      "sealed",
			"event_id":    evt.ID,
			"leaf_hash":   leafHash,
			"block_index": block.Header.Index,
			"leaf_index":  evt.LeafIndex,
			"received_at": evt.Timestamp.UTC().Format(time.RFC3339Nano),
		})
		return
	}

	leafHash, err := h.acc.IngestAsync(&evt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("async ingestion error: %v", err))
		return
	}

	// Broadcast live to connected clients
	h.sseHub.BroadcastEvent("event_received", map[string]any{
		"event_id":   evt.ID,
		"action":     evt.Action,
		"actor_name": evt.Actor.Name,
		"tenant_id":  evt.TenantID,
		"timestamp":  evt.Timestamp.UTC().Format(time.RFC3339Nano),
		"leaf_hash":  leafHash,
	})

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      "accepted",
		"event_id":    evt.ID,
		"leaf_hash":   leafHash,
		"received_at": evt.Timestamp.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) handleIngestBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID string                `json:"tenant_id"`
		Events   []*ledger.AuditEvent `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid batch json: %v", err))
		return
	}

	if len(req.Events) == 0 {
		writeJSONError(w, http.StatusBadRequest, "empty events array")
		return
	}

	leafHashes := make([]string, 0, len(req.Events))
	for _, evt := range req.Events {
		if evt.TenantID == "" {
			evt.TenantID = req.TenantID
		}
		leafHash, err := h.acc.IngestAsync(evt)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed ingesting event %s: %v", evt.ID, err))
			return
		}
		leafHashes = append(leafHashes, leafHash)
	}

	if r.URL.Query().Get("sync") == "true" {
		if err := h.acc.Flush(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed flushing batch: %v", err))
			return
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":         "accepted",
		"ingested_count": len(req.Events),
		"leaf_hashes":    leafHashes,
	})
}

func (h *Handler) handleSearchEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := index.EventFilter{
		TenantID:   q.Get("tenant_id"),
		ActorID:    q.Get("actor_id"),
		Action:     q.Get("action"),
		TargetType: q.Get("target_type"),
		TargetID:   q.Get("target_id"),
		Query:      q.Get("q"),
	}

	if bi := q.Get("block_index"); bi != "" {
		if parsed, err := strconv.ParseUint(bi, 10, 64); err == nil {
			filter.BlockIndex = &parsed
		}
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		filter.Limit, _ = strconv.Atoi(limitStr)
	}
	if offsetStr := q.Get("offset"); offsetStr != "" {
		filter.Offset, _ = strconv.Atoi(offsetStr)
	}

	results, err := h.indexer.SearchEvents(r.Context(), filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("query error: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) handleGetEventByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "event id is required")
		return
	}

	evt, err := h.indexer.GetEventByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("database error: %v", err))
		return
	}
	if evt == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("event %s not found", id))
		return
	}

	// Compute inclusion proof from block
	block, err := h.indexer.GetBlockByIndex(r.Context(), evt.BlockIndex)
	if err != nil || block == nil {
		writeJSON(w, http.StatusOK, map[string]any{"event": evt})
		return
	}

	leaves := make([][]byte, len(block.Events))
	for i, e := range block.Events {
		lh, _ := e.ComputeLeafHash(h.algo)
		leaves[i] = lh
	}
	tree := ledger.NewMerkleTree(h.algo, leaves)
	proof, _ := tree.GetProof(evt.LeafIndex)

	writeJSON(w, http.StatusOK, map[string]any{
		"event":        evt,
		"block_header": block.Header,
		"proof":        proof,
		"verified":     ledger.VerifyMerkleProof(proof),
	})
}

func (h *Handler) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	res, err := h.indexer.ListBlocks(r.Context(), limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed listing blocks: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) handleGetBlockByIndex(w http.ResponseWriter, r *http.Request) {
	rawIndex := strings.TrimPrefix(r.URL.Path, "/api/v1/blocks/")
	idx, err := strconv.ParseUint(rawIndex, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid block index format")
		return
	}

	block, err := h.indexer.GetBlockByIndex(r.Context(), idx)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed querying block: %v", err))
		return
	}
	if block == nil {
		writeJSONError(w, http.StatusNotFound, fmt.Sprintf("block %d not found", idx))
		return
	}

	writeJSON(w, http.StatusOK, block)
}

func (h *Handler) handleVerifyLedger(w http.ResponseWriter, r *http.Request) {
	blocks, err := h.wal.ReadAllBlocks()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed reading wal: %v", err))
		return
	}

	// If simulation active, inject corruption into that block for demonstration
	if h.simulatedTamperBlock != nil && len(blocks) > int(*h.simulatedTamperBlock) {
		idx := *h.simulatedTamperBlock
		blocks[idx].Events[0].Action = "tampered.unauthorized.action"
		_, _ = blocks[idx].Events[0].ComputeLeafHash(h.algo)
	}

	res := ledger.VerifyLedgerChain(blocks)
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.indexer.GetLedgerStats(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed fetching stats: %v", err))
		return
	}

	blocks, _ := h.wal.ReadAllBlocks()
	var chainValid = true
	var headHash string
	if len(blocks) > 0 {
		res := ledger.VerifyLedgerChain(blocks)
		chainValid = res.Valid
		headHash = hex.EncodeToString(blocks[len(blocks)-1].Header.BlockHash)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_events":      stats.TotalEvents,
		"total_blocks":      stats.TotalBlocks,
		"tenant_count":      stats.TenantCount,
		"first_event_time":  stats.FirstEventTime,
		"last_event_time":   stats.LastEventTime,
		"ledger_verified":   chainValid,
		"head_block_hash":   headHash,
		"algorithm":         h.algo,
		"tamper_simulation": h.simulatedTamperBlock != nil,
	})
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	stats, _ := h.indexer.GetLedgerStats(r.Context())
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP auditforge_events_total Total number of indexed audit events\n")
	fmt.Fprintf(w, "# TYPE auditforge_events_total counter\n")
	fmt.Fprintf(w, "auditforge_events_total %d\n", stats.TotalEvents)

	fmt.Fprintf(w, "# HELP auditforge_blocks_total Total number of sealed cryptographic blocks\n")
	fmt.Fprintf(w, "# TYPE auditforge_blocks_total counter\n")
	fmt.Fprintf(w, "auditforge_blocks_total %d\n", stats.TotalBlocks)
}

func (h *Handler) handleSimulateTamper(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action     string  `json:"action"` // "enable" or "disable"
		BlockIndex *uint64 `json:"block_index"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Action == "disable" {
		h.simulatedTamperBlock = nil
		writeJSON(w, http.StatusOK, map[string]any{"status": "tamper_simulation_disabled"})
		return
	}

	var target uint64 = 0
	if req.BlockIndex != nil {
		target = *req.BlockIndex
	}
	h.simulatedTamperBlock = &target

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                 "tamper_simulation_enabled",
		"simulated_target_block": target,
		"message":                "Simulated tamper injected. Run verify or inspect UI to observe immediate shield violation alert.",
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
