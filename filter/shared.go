package filter

import (
	"encoding/binary"
)

// Packet represents a captured network packet.
// Addr holds the WinDivert address as raw bytes ([]byte) when built with windivert tag.
type Packet struct {
	Data    []byte
	Addr    interface{}
	SrcIP   uint32
	DstIP   uint32
	SrcPort uint16
	DstPort uint16
	Proto   uint8
}

// isImpostor checks the Impostor bit in a WINDIVERT_ADDRESS byte slice.
// The Impostor flag (bit 19) is set by WinDivert for re-injected packets.
func isImpostor(addr interface{}) bool {
	b, ok := addr.([]byte)
	if !ok || len(b) < 12 {
		return false
	}
	// Flags UINT32 is at offset 8 (after INT64 Timestamp) in WINDIVERT_ADDRESS.
	// Impostor = bit 19: Layer(0-7) + Event(8-15) + Sniffed(16) + Outbound(17) + Loopback(18) + Impostor(19)
	flags := binary.LittleEndian.Uint32(b[8:12])
	return (flags>>19)&1 == 1
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
