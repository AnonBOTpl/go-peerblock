package core

import (
	"net"
	"strings"
	"sync"
	"time"
)

// Allowlist holds IP ranges that should never be blocked.
type Allowlist struct {
	staticRanges []IPRange
	domains      []string
	ranges       []IPRange
	mu           sync.RWMutex
}

// NewAllowlist creates an Allowlist from raw string entries.
// Each entry can be an IP, CIDR range, or domain name.
func NewAllowlist(entries []string) *Allowlist {
	a := &Allowlist{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}
		if strings.Contains(entry, "/") {
			if r, err := CIDRToRange(entry); err == nil {
				a.staticRanges = append(a.staticRanges, r)
			}
		} else if strings.Contains(entry, "*") || strings.Contains(entry, ".") && !isIPString(entry) {
			a.domains = append(a.domains, entry)
		} else if ip := net.ParseIP(entry).To4(); ip != nil {
			n := IPToUint32(ip)
			a.staticRanges = append(a.staticRanges, IPRange{Start: n, End: n})
		}
	}
	a.staticRanges = MergeRanges(a.staticRanges)
	a.ranges = a.staticRanges
	return a
}

// Contains checks if an IP is in the allowlist.
func (a *Allowlist) Contains(ip uint32) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ranges := a.ranges
	lo, hi := 0, len(ranges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		r := ranges[mid]
		if ip < r.Start {
			hi = mid - 1
		} else if ip > r.End {
			lo = mid + 1
		} else {
			return true
		}
	}
	return false
}

// ResolveAndRefresh resolves all domain entries and rebuilds the range list.
func (a *Allowlist) ResolveAndRefresh() {
	var newRanges []IPRange
	newRanges = append(newRanges, a.staticRanges...)

	for _, domain := range a.domains {
		// Handle wildcard: resolve the base domain
		d := strings.TrimPrefix(domain, "*.")
		addrs, err := net.LookupHost(d)
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := net.ParseIP(addr).To4()
			if ip == nil {
				continue
			}
			n := IPToUint32(ip)
			newRanges = append(newRanges, IPRange{Start: n, End: n})
		}
	}

	merged := MergeRanges(newRanges)
	a.mu.Lock()
	a.ranges = merged
	a.mu.Unlock()
}

// StartRefreshLoop refreshes DNS entries periodically.
func (a *Allowlist) StartRefreshLoop(interval time.Duration, done chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.ResolveAndRefresh()
		case <-done:
			return
		}
	}
}

func isIPString(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

