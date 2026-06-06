package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
)

// Format represents the format of an IP blocklist file.
type Format int

const (
	FormatUnknown Format = iota
	FormatP2PText        // PeerGuardian Text (.p2p): "Level1:1.2.3.0-1.2.3.255"
	FormatDAT            // eMule DAT: "1.2.3.0 - 1.2.3.255 , 100 , Blocklist"
	FormatCIDR           // Plain CIDR: "1.2.3.0/24" one per line
	FormatPlainRange     // Plain range: "1.2.3.0-1.2.3.255"
)

// FormatDetector auto-detects the format of IP list data.
type FormatDetector struct{}

// Detect reads the first 512 bytes to determine the format.
func (d FormatDetector) Detect(r io.Reader) (Format, io.Reader, error) {
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	header := buf[:n]
	combined := io.MultiReader(bytes.NewReader(header), r)

	switch {
	case bytes.Contains(header, []byte(" - ")) && bytes.Contains(header, []byte(" , ")):
		return FormatDAT, combined, nil
	case bytes.Contains(header, []byte(":")):
		return FormatP2PText, combined, nil
	default:
		return FormatCIDR, combined, nil
	}
}

// Parse parses IP ranges from a byte slice in the given format.
func Parse(data []byte, format Format) ([]IPRange, error) {
	switch format {
	case FormatP2PText:
		return parseP2PText(data)
	case FormatDAT:
		return parseDAT(data)
	case FormatCIDR:
		return parseCIDR(data)
	case FormatPlainRange:
		return parsePlainRange(data)
	default:
		return parseCIDR(data)
	}
}

func parseP2PText(data []byte) ([]IPRange, error) {
	var ranges []IPRange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "Level1:1.2.3.0-1.2.3.255"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, err := parseRangeStr(parts[1])
		if err != nil {
			return nil, fmt.Errorf("p2p parse error: %w", err)
		}
		r.Label = parts[0]
		ranges = append(ranges, r)
	}
	return ranges, scanner.Err()
}

func parseDAT(data []byte) ([]IPRange, error) {
	var ranges []IPRange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "1.2.3.0 - 1.2.3.255 , 100 , Blocklist"
		parts := strings.SplitN(line, ",", 3)
		if len(parts) < 2 {
			continue
		}
		r, err := parseRangeStr(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		ranges = append(ranges, r)
	}
	return ranges, scanner.Err()
}

func parseCIDR(data []byte) ([]IPRange, error) {
	var ranges []IPRange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		r, err := CIDRToRange(line)
		if err != nil {
			continue
		}
		ranges = append(ranges, r)
	}
	return ranges, scanner.Err()
}

func parsePlainRange(data []byte) ([]IPRange, error) {
	var ranges []IPRange
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r, err := parseRangeStr(line)
		if err != nil {
			continue
		}
		ranges = append(ranges, r)
	}
	return ranges, scanner.Err()
}

func parseRangeStr(s string) (IPRange, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return IPRange{}, fmt.Errorf("invalid range format: %s", s)
	}
	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))
	if startIP == nil || endIP == nil {
		return IPRange{}, fmt.Errorf("invalid IP in range: %s", s)
	}
	return IPRange{
		Start: IPToUint32(startIP),
		End:   IPToUint32(endIP),
	}, nil
}
