//go:build !windivert

package filter

import (
	"time"

	"go-peerblock/core"
)

// Pipeline is a no-op stub when built without the windivert tag.
type Pipeline struct {
	started   bool
	startedAt int64
}

// NewPipeline creates a no-op pipeline (no WinDivert available).
func NewPipeline(wd *WinDivert, db *core.IPDatabase, cache *core.DecisionCache, allowlist *core.Allowlist, workerCount int) *Pipeline {
	return &Pipeline{}
}

// Start is a no-op in stub mode.
func (p *Pipeline) Start() {
	p.started = true
	p.startedAt = time.Now().UnixNano()
}

// Stop is a no-op in stub mode.
func (p *Pipeline) Stop() {
	p.started = false
}

// Close is a no-op in stub mode.
func (p *Pipeline) Close() {
	p.started = false
}

// IsRunning returns the started state.
func (p *Pipeline) IsRunning() bool {
	return p.started
}

// GetStats returns basic stats with start time.
func (p *Pipeline) GetStats() Stats {
	return Stats{StartedAt: p.startedAt}
}

// shouldBlock is a no-op in stub mode.
func (p *Pipeline) shouldBlock(pkt Packet) bool {
	return false
}
