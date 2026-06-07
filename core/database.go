package core

import (
	"encoding/binary"
	"net"
	"sort"
	"sync/atomic"
)

// IPRange represents a range of IP addresses (inclusive).
type IPRange struct {
	Start uint32
	End   uint32
	Label string
}

// IPDatabase holds a sorted, non-overlapping list of IP ranges.
type IPDatabase struct {
	ranges atomic.Pointer[[]IPRange]
}

// NewDatabase creates a new IPDatabase from the given ranges.
func NewDatabase(ranges []IPRange) *IPDatabase {
	db := &IPDatabase{}
	merged := MergeRanges(ranges)
	db.ranges.Store(&merged)
	return db
}

// Ranges returns the current list of ranges (read-only).
func (db *IPDatabase) Ranges() []IPRange {
	return *db.ranges.Load()
}

// Store atomically replaces the ranges.
func (db *IPDatabase) Store(ranges []IPRange) {
	merged := MergeRanges(ranges)
	db.ranges.Store(&merged)
}

// Contains checks if an IP (as uint32) is in any of the ranges using binary search.
func (db *IPDatabase) Contains(ip uint32) bool {
	ranges := *db.ranges.Load()
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

// MergeRanges sorts and merges overlapping or adjacent IP ranges.
func MergeRanges(ranges []IPRange) []IPRange {
	if len(ranges) == 0 {
		return nil
	}

	sorted := make([]IPRange, len(ranges))
	copy(sorted, ranges)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start < sorted[j].Start
	})

	merged := []IPRange{sorted[0]}
	for _, r := range sorted[1:] {
		last := &merged[len(merged)-1]
		if r.Start <= last.End+1 {
			if r.End > last.End {
				last.End = r.End
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// CIDRToRange converts a CIDR string (e.g. "1.2.3.0/24") to an IPRange.
func CIDRToRange(cidr string) (IPRange, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return IPRange{}, err
	}
	ip := network.IP.To4()
	if ip == nil {
		return IPRange{}, net.InvalidAddrError("not an IPv4 address")
	}
	start := binary.BigEndian.Uint32(ip)
	mask := binary.BigEndian.Uint32(network.Mask)
	end := start | ^mask
	return IPRange{Start: start, End: end}, nil
}

// IPToUint32 converts a net.IP (IPv4) to uint32.
func IPToUint32(ip net.IP) uint32 {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip4)
}

// Uint32ToIP converts a uint32 back to dotted-decimal IPv4 string (e.g. "1.2.3.4").
func Uint32ToIP(n uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip.String()
}
