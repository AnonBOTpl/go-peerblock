//go:build windivert

package filter

import (
	"sync/atomic"
	"time"

	"go-peerblock/core"
)

// BlockCallback is called when a packet is blocked, with source IP, destination IP, and protocol.
type BlockCallback func(srcIP, dstIP uint32, proto uint8)

// Pipeline processes packets through a multi-worker pipeline.
type Pipeline struct {
	wd          *WinDivert
	db          *atomic.Pointer[core.IPDatabase]
	cache       *core.DecisionCache
	allowlist   *core.Allowlist
	packetCh    chan Packet
	sendCh      chan Packet
	workerCount int
	allowed     atomic.Uint64
	blocked     atomic.Uint64
	dropped     atomic.Uint64
	startedAt   int64
	done        chan struct{}
	started     atomic.Bool
	onBlock     BlockCallback
}

// NewPipeline creates a new packet processing pipeline.
func NewPipeline(wd *WinDivert, db *atomic.Pointer[core.IPDatabase], cache *core.DecisionCache, allowlist *core.Allowlist, workerCount int) *Pipeline {
	if workerCount <= 0 {
		workerCount = 4
	}

	return &Pipeline{
		wd:          wd,
		db:          db,
		cache:       cache,
		allowlist:   allowlist,
		packetCh:    make(chan Packet, 4096),
		sendCh:      make(chan Packet, 4096),
		workerCount: workerCount,
		startedAt:   time.Now().UnixNano(),
		done:        make(chan struct{}),
	}
}

// Close shuts down the pipeline and closes the WinDivert handle.
func (p *Pipeline) Close() {
	p.Stop()
	if p.wd != nil {
		p.wd.Close()
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
	// Close WinDivert handle first to unblock any pending Recv() call,
	// then signal goroutines via done channel.
	if p.wd != nil && p.wd.IsOpen() {
		p.wd.Close()
	}
	close(p.done)
}

// IsRunning returns whether the pipeline is active.
func (p *Pipeline) IsRunning() bool {
	return p.started.Load()
}

// SetOnBlock registers a callback invoked when a packet is blocked.
func (p *Pipeline) SetOnBlock(fn BlockCallback) {
	p.onBlock = fn
}

// GetStats returns a copy of the current stats.
func (p *Pipeline) GetStats() Stats {
	return Stats{
		Allowed:   p.allowed.Load(),
		Blocked:   p.blocked.Load(),
		Dropped:   p.dropped.Load(),
		StartedAt: p.startedAt,
	}
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

		// Impostor flag = packet was re-injected by us. Send it back immediately
		// to prevent infinite loop (recv → process → send → recv → ...).
		if isImpostor(addr) {
			_, err := p.wd.Send(buf[:n], addr)
			if err != nil {
				p.dropped.Add(1)
			}
			continue
		}

		pkt := Packet{Data: make([]byte, n)}
		pkt.Addr = addr
		copy(pkt.Data, buf[:n])
		pkt.SrcIP, pkt.DstIP, pkt.Proto = ParseIPHeader(pkt.Data)

		select {
		case p.packetCh <- pkt:
		default:
			p.dropped.Add(1)
		}
	}
}

func (p *Pipeline) worker() {
	for {
		select {
		case <-p.done:
			return
		case pkt, ok := <-p.packetCh:
			if !ok {
				return
			}
			if p.shouldBlock(pkt) {
				p.blocked.Add(1)
				if p.onBlock != nil {
					p.onBlock(pkt.SrcIP, pkt.DstIP, pkt.Proto)
				}
				continue
			}
			p.allowed.Add(1)
			select {
			case <-p.done:
				return
			case p.sendCh <- pkt:
			}
		}
	}
}

func (p *Pipeline) sendLoop() {
	for {
		select {
		case <-p.done:
			return
		case pkt, ok := <-p.sendCh:
			if !ok {
				return
			}
			_, err := p.wd.Send(pkt.Data, pkt.Addr)
			if err != nil {
				p.dropped.Add(1)
			}
		}
	}
}

func (p *Pipeline) shouldBlock(pkt Packet) bool {
	// Only check destination IP — source IP is the user's local IP
	// (e.g. 192.168.x.x, 172.16.x.x, 10.x.x.x) and was never meant to bypass blocking.
	if p.allowlist.Contains(pkt.DstIP) {
		return false
	}

	ip := pkt.DstIP
	if blocked, ok := p.cache.Get(ip); ok {
		return blocked
	}

	db := p.db.Load()
	if db == nil {
		return false
	}
	blocked := db.Contains(ip)
	p.cache.Set(ip, blocked)
	return blocked
}
