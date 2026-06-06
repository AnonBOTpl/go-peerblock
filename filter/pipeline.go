//go:build windivert

package filter

import (
	"sync/atomic"
	"time"

	"go-peerblock/core"
)

// Pipeline processes packets through a multi-worker pipeline.
type Pipeline struct {
	wd          *WinDivert
	db          *core.IPDatabase
	cache       *core.DecisionCache
	allowlist   *core.Allowlist
	packetCh    chan Packet
	sendCh      chan Packet
	workerCount int
	stats       atomic.Value
	done        chan struct{}
	started     atomic.Bool
}

// NewPipeline creates a new packet processing pipeline.
func NewPipeline(wd *WinDivert, db *core.IPDatabase, cache *core.DecisionCache, allowlist *core.Allowlist, workerCount int) *Pipeline {
	if workerCount <= 0 {
		workerCount = 4
	}
	stats := Stats{StartedAt: time.Now().UnixNano()}
	var sv atomic.Value
	sv.Store(stats)

	return &Pipeline{
		wd:          wd,
		db:          db,
		cache:       cache,
		allowlist:   allowlist,
		packetCh:    make(chan Packet, 4096),
		sendCh:      make(chan Packet, 4096),
		workerCount: workerCount,
		done:        make(chan struct{}),
	}
}

// Start launches the pipeline goroutines.
func (p *Pipeline) Start() {
	if p.started.Load() {
		return
	}
	p.started.Store(true)

	go p.recvLoop()
	for i := 0; i < p.workerCount; i++ {
		go p.worker()
	}
	go p.sendLoop()
}

// Stop gracefully shuts down the pipeline.
func (p *Pipeline) Stop() {
	if !p.started.Load() {
		return
	}
	p.started.Store(false)
	close(p.done)
}

// IsRunning returns whether the pipeline is active.
func (p *Pipeline) IsRunning() bool {
	return p.started.Load()
}

// GetStats returns a copy of the current stats.
func (p *Pipeline) GetStats() Stats {
	return p.stats.Load().(Stats)
}

func (p *Pipeline) recvLoop() {
	buf := make([]byte, 65535)
	for {
		select {
		case <-p.done:
			return
		default:
		}

		n, addr, err := p.wd.Recv(buf)
		if err != nil {
			select {
			case <-p.done:
				return
			default:
				continue
			}
		}

		pkt := Packet{Data: make([]byte, n)}
		pkt.Addr = addr
		copy(pkt.Data, buf[:n])
		pkt.SrcIP, pkt.DstIP, pkt.Proto = ParseIPHeader(pkt.Data)

		select {
		case p.packetCh <- pkt:
		default:
			s := p.stats.Load().(Stats)
			s.Dropped++
			p.stats.Store(s)
		}
	}
}

func (p *Pipeline) worker() {
	for pkt := range p.packetCh {
		if p.shouldBlock(pkt) {
			s := p.stats.Load().(Stats)
			s.Blocked++
			p.stats.Store(s)
			continue
		}
		s := p.stats.Load().(Stats)
		s.Allowed++
		p.stats.Store(s)
		p.sendCh <- pkt
	}
}

func (p *Pipeline) sendLoop() {
	for pkt := range p.sendCh {
		_, err := p.wd.Send(pkt.Data, pkt.Addr)
		if err != nil {
			s := p.stats.Load().(Stats)
			s.Dropped++
			p.stats.Store(s)
		}
	}
}

func (p *Pipeline) shouldBlock(pkt Packet) bool {
	if p.allowlist.Contains(pkt.SrcIP) || p.allowlist.Contains(pkt.DstIP) {
		return false
	}
	db := p.db
	if db == nil {
		return false
	}
	for _, ip := range []uint32{pkt.SrcIP, pkt.DstIP} {
		if blocked, ok := p.cache.Get(ip); ok {
			if blocked {
				return true
			}
			continue
		}
		blocked := db.Contains(ip)
		p.cache.Set(ip, blocked)
		if blocked {
			return true
		}
	}
	return false
}
