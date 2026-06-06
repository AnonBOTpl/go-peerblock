package filter

import (
	"encoding/binary"
)

// Packet represents a captured network packet.
// Addr holds the WinDivert address (gowindivert.Address when built with windivert tag).
type Packet struct {
	Data    []byte
	Addr    interface{}
	SrcIP   uint32
	DstIP   uint32
	SrcPort uint16
	DstPort uint16
	Proto   uint8
}

// Stats holds pipeline statistics.
// StartedAt is UnixNano timestamp (int64) for clean JSON serialization.
type Stats struct {
	Allowed   uint64 `json:"allowed"`
	Blocked   uint64 `json:"blocked"`
	Dropped   uint64 `json:"dropped"`
	StartedAt int64  `json:"started_at"`
}

// ParseIPHeader extracts source IP, destination IP, and protocol from an IPv4 header.
func ParseIPHeader(data []byte) (srcIP, dstIP uint32, proto uint8) {
	if len(data) < 20 || (data[0]>>4) != 4 {
		return 0, 0, 0
	}
	proto = data[9]
	srcIP = binary.BigEndian.Uint32(data[12:16])
	dstIP = binary.BigEndian.Uint32(data[16:20])
	return
}

// DefaultFilter returns the recommended WinDivert filter string.
func DefaultFilter() string {
	return "ip and (ip.DstAddr != 127.0.0.1) and (ip.SrcAddr != 127.0.0.1)"
}
