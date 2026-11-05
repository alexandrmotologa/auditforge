package ledger

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BlockHandler is a callback invoked whenever a new block is sealed and committed
type BlockHandler func(block *Block, tree *MerkleTree) error

// AccumulatorConfig sets the batching parameters for the ledger accumulator
type AccumulatorConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	Algorithm     string
}

// DefaultAccumulatorConfig returns optimal production settings (500 events or 100ms)
func DefaultAccumulatorConfig() AccumulatorConfig {
	return AccumulatorConfig{
		BatchSize:     500,
		FlushInterval: 100 * time.Millisecond,
		Algorithm:     AlgorithmBlake3,
	}
}

// syncRequest wraps an event with a channel waiting for seal completion
type syncRequest struct {
	event *AuditEvent
	done  chan error
	block *Block
}

// Accumulator coordinates in-memory event batching, block sealing, and WAL persistence
type Accumulator struct {
	mu           sync.Mutex
	cfg          AccumulatorConfig
	wal          *WAL
	signer       Signer
	onBlock      BlockHandler
	pending      []*AuditEvent
	syncWaiters  []*syncRequest
	lastHash     []byte
	nextIndex    uint64
	ticker       *time.Ticker
	quit         chan struct{}
	flushTrigger chan struct{}
}

// NewAccumulator initializes the accumulator, resuming sequence index and previous hash from WAL
func NewAccumulator(cfg AccumulatorConfig, wal *WAL, signer Signer, onBlock BlockHandler) (*Accumulator, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 500
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 100 * time.Millisecond
	}

	existingBlocks, err := wal.ReadAllBlocks()
	if err != nil {
		return nil, fmt.Errorf("failed reading existing blocks from wal: %w", err)
	}

	var nextIdx uint64 = 0
	var lastHash []byte = make([]byte, 32) // Genesis block previous hash

	if len(existingBlocks) > 0 {
		lastBlock := existingBlocks[len(existingBlocks)-1]
		nextIdx = lastBlock.Header.Index + 1
		lastHash = lastBlock.Header.BlockHash
	}

	acc := &Accumulator{
		cfg:          cfg,
		wal:          wal,
		signer:       signer,
		onBlock:      onBlock,
		pending:      make([]*AuditEvent, 0, cfg.BatchSize),
		syncWaiters:  make([]*syncRequest, 0),
		lastHash:     lastHash,
		nextIndex:    nextIdx,
		ticker:       time.NewTicker(cfg.FlushInterval),
		quit:         make(chan struct{}),
		flushTrigger: make(chan struct{}, 1),
	}

	go acc.loop()

	return acc, nil
}

// IngestAsync accepts an event for batched sealing, returning immediately with event ID and leaf hash
func (a *Accumulator) IngestAsync(evt *AuditEvent) (string, error) {
	if evt.ID == "" {
		evt.ID = NewEventID()
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	leafHash, err := evt.ComputeLeafHash(a.cfg.Algorithm)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	a.pending = append(a.pending, evt)
	shouldFlush := len(a.pending) >= a.cfg.BatchSize
	a.mu.Unlock()

	if shouldFlush {
		select {
		case a.flushTrigger <- struct{}{}:
		default:
		}
	}

	return FormatHashHex(leafHash), nil
}

// IngestSync ingests an event and blocks until the event is sealed in a durable block
func (a *Accumulator) IngestSync(ctx context.Context, evt *AuditEvent) (*Block, string, error) {
	if evt.ID == "" {
		evt.ID = NewEventID()
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	leafHash, err := evt.ComputeLeafHash(a.cfg.Algorithm)
	if err != nil {
		return nil, "", err
	}

	req := &syncRequest{
		event: evt,
		done:  make(chan error, 1),
	}

	a.mu.Lock()
	a.pending = append(a.pending, evt)
	a.syncWaiters = append(a.syncWaiters, req)
	shouldFlush := len(a.pending) >= a.cfg.BatchSize
	a.mu.Unlock()

	if shouldFlush {
		select {
		case a.flushTrigger <- struct{}{}:
		default:
		}
	}

	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case err := <-req.done:
		if err != nil {
			return nil, "", err
		}
		return req.block, FormatHashHex(leafHash), nil
	}
}

// Flush forces immediate sealing and persistence of all pending events
func (a *Accumulator) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.flushCurrentBatchLocked()
}

func (a *Accumulator) loop() {
	for {
		select {
		case <-a.quit:
			return
		case <-a.ticker.C:
			a.mu.Lock()
			if len(a.pending) > 0 {
				_ = a.flushCurrentBatchLocked()
			}
			a.mu.Unlock()
		case <-a.flushTrigger:
			a.mu.Lock()
			if len(a.pending) > 0 {
				_ = a.flushCurrentBatchLocked()
			}
			a.mu.Unlock()
		}
	}
}

func (a *Accumulator) flushCurrentBatchLocked() error {
	if len(a.pending) == 0 {
		return nil
	}

	events := a.pending
	waiters := a.syncWaiters

	a.pending = make([]*AuditEvent, 0, a.cfg.BatchSize)
	a.syncWaiters = make([]*syncRequest, 0)

	block, tree, err := NewBlock(a.nextIndex, a.lastHash, events, a.cfg.Algorithm)
	if err != nil {
		for _, w := range waiters {
			w.done <- err
		}
		return fmt.Errorf("failed creating block %d: %w", a.nextIndex, err)
	}

	if err := block.Seal(a.signer); err != nil {
		for _, w := range waiters {
			w.done <- err
		}
		return fmt.Errorf("failed sealing block %d: %w", a.nextIndex, err)
	}

	// Commit to WAL
	if _, err := a.wal.AppendBlock(block); err != nil {
		for _, w := range waiters {
			w.done <- err
		}
		return fmt.Errorf("failed writing block %d to wal: %w", a.nextIndex, err)
	}

	// Update sequence state
	a.lastHash = block.Header.BlockHash
	a.nextIndex++

	// Notify downstream callback (e.g., secondary SQLite index and SSE broadcast)
	if a.onBlock != nil {
		if err := a.onBlock(block, tree); err != nil {
			// Log error but keep ledger intact
			fmt.Printf("warning: onBlock callback error on block %d: %v\n", block.Header.Index, err)
		}
	}

	// Notify all synchronous callers
	for _, w := range waiters {
		w.block = block
		w.done <- nil
	}

	return nil
}

// Close gracefully flushes all remaining events and terminates background ticker
func (a *Accumulator) Close() error {
	close(a.quit)
	a.ticker.Stop()
	return a.Flush()
}
