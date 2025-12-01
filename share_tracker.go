package main

import (
	"context"
	"encoding/hex"
	"sync"
	"time"
	"xelis-mining-proxy/log"
	"xelis-mining-proxy/stratum"

	"github.com/gorilla/websocket"
)

// PendingShare represents a share waiting for pool response
type PendingShare struct {
	RequestID     interface{}  // Stratum request ID to respond with (nil for getwork)
	StratumConn   *StratumConn // Stratum connection to send response to (nil for getwork)
	GetworkConn   *GetworkConn // Getwork connection to send response to (nil for stratum)
	SubmittedAt   time.Time    // When the share was submitted
	ResponseChan  chan ShareResult
	CancelFunc    context.CancelFunc // To cancel the timeout goroutine
}

// ShareResult contains the pool's response for a share
type ShareResult struct {
	Accepted bool
	Error    *stratum.Error
}

// ShareTracker manages pending shares awaiting pool responses
type ShareTracker struct {
	mu            sync.RWMutex
	pendingShares map[string]*PendingShare // Key: hex(extra_nonce + nonce)
	timeout       time.Duration
}

// NewShareTracker creates a new share tracker with specified timeout
func NewShareTracker(timeout time.Duration) *ShareTracker {
	return &ShareTracker{
		pendingShares: make(map[string]*PendingShare),
		timeout:       timeout,
	}
}

// GenerateShareID creates a unique share ID from extra nonce and nonce
// This is guaranteed unique because extra nonce is unique per miner connection
// and nonce is unique per share attempt from that miner
func GenerateShareID(extraNonce [32]byte, nonce [8]byte) string {
	// Concatenate extra nonce (32 bytes) + nonce (8 bytes) = 40 bytes
	combined := make([]byte, 40)
	copy(combined[0:32], extraNonce[:])
	copy(combined[32:40], nonce[:])
	return hex.EncodeToString(combined)
}

// ExtractShareID extracts the share ID from a complete block (112 bytes)
// Block structure: workhash(32) + timestamp(8) + nonce(8) + extra_nonce(32) + pubkey(32)
func ExtractShareID(block []byte) (string, error) {
	if len(block) != 112 {
		log.Warnf("ExtractShareID: block length %d != 112", len(block))
		return "", nil
	}

	// Extract nonce (bytes 40-48) and extra nonce (bytes 48-80)
	nonce := [8]byte(block[40:48])
	extraNonce := [32]byte(block[48:80])

	return GenerateShareID(extraNonce, nonce), nil
}

// AddPendingShare registers a share awaiting pool response
func (st *ShareTracker) AddPendingShare(shareID string, pending *PendingShare) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.pendingShares[shareID] = pending
	log.Debugf("Added pending share %s (total pending: %d)", shareID, len(st.pendingShares))
}

// RemovePendingShare removes a share from tracking
func (st *ShareTracker) RemovePendingShare(shareID string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if pending, exists := st.pendingShares[shareID]; exists {
		// Cancel the timeout goroutine if it exists
		if pending.CancelFunc != nil {
			pending.CancelFunc()
		}
		delete(st.pendingShares, shareID)
		log.Debugf("Removed pending share %s (total pending: %d)", shareID, len(st.pendingShares))
	}
}

// GetPendingShare retrieves a pending share by ID
func (st *ShareTracker) GetPendingShare(shareID string) *PendingShare {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.pendingShares[shareID]
}

// StartResponseWaiter starts a goroutine that waits for pool response or timeout
func (st *ShareTracker) StartResponseWaiter(shareID string, pending *PendingShare) {
	ctx, cancel := context.WithTimeout(context.Background(), st.timeout)
	pending.CancelFunc = cancel

	go func() {
		defer cancel()
		defer st.RemovePendingShare(shareID)

		select {
		case result := <-pending.ResponseChan:
			// Got pool response - send to miner based on connection type
			log.Debugf("Share %s: sending result (accepted=%v) to miner", shareID, result.Accepted)

			if pending.StratumConn != nil {
				// Stratum response
				pending.StratumConn.Lock()
				defer pending.StratumConn.Unlock()

				if !pending.StratumConn.Alive {
					log.Debugf("Share %s: stratum connection closed, skipping response", shareID)
					return
				}

				err := pending.StratumConn.WriteJSON(stratum.ResponseOut{
					Id:     pending.RequestID.(uint32),
					Result: result.Accepted,
					Error:  result.Error,
				})
				if err != nil {
					log.Warnf("Share %s: failed to send stratum response: %v", shareID, err)
				}
			} else if pending.GetworkConn != nil {
				// Getwork response
				pending.GetworkConn.Lock()
				defer pending.GetworkConn.Unlock()

				var msg string
				if result.Accepted {
					msg = `"block_accepted"`
				} else {
					msg = `"block_rejected"`
				}

				err := pending.GetworkConn.conn.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					log.Warnf("Share %s: failed to send getwork response: %v", shareID, err)
				}
			}

		case <-ctx.Done():
			// Timeout - send rejection to miner
			log.Warnf("Share %s timed out after %v waiting for pool response", shareID, st.timeout)

			if pending.StratumConn != nil {
				pending.StratumConn.Lock()
				defer pending.StratumConn.Unlock()

				if !pending.StratumConn.Alive {
					log.Debugf("Share %s: stratum connection closed during timeout", shareID)
					return
				}

				err := pending.StratumConn.WriteJSON(stratum.ResponseOut{
					Id:     pending.RequestID.(uint32),
					Result: false,
					Error: &stratum.Error{
						Code:    -1,
						Message: "pool response timeout",
					},
				})
				if err != nil {
					log.Warnf("Share %s: failed to send timeout response: %v", shareID, err)
				}
			} else if pending.GetworkConn != nil {
				pending.GetworkConn.Lock()
				defer pending.GetworkConn.Unlock()

				err := pending.GetworkConn.conn.WriteMessage(websocket.TextMessage, []byte(`"block_rejected"`))
				if err != nil {
					log.Warnf("Share %s: failed to send getwork timeout response: %v", shareID, err)
				}
			}
		}
	}()
}

// GetPendingCount returns the number of shares awaiting responses
func (st *ShareTracker) GetPendingCount() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.pendingShares)
}
