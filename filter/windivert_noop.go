//go:build !windivert

package filter

import (
	"fmt"
	"sync/atomic"
)

// WinDivert is a no-op implementation when WinDivert is not available.
type WinDivert struct {
	open atomic.Bool
}

// Open creates a no-op WinDivert handle.
func Open(filter string, layer interface{}, priority int16) (*WinDivert, error) {
	return &WinDivert{}, nil
}

// Recv is not supported in noop mode.
func (w *WinDivert) Recv(buf []byte) (int, interface{}, error) {
	return 0, nil, fmt.Errorf("WinDivert not available")
}

// Send is not supported in noop mode.
func (w *WinDivert) Send(buf []byte, addr interface{}) (int, error) {
	return 0, fmt.Errorf("WinDivert not available")
}

// Close is a no-op.
func (w *WinDivert) Close() error {
	w.open.Store(false)
	return nil
}

// IsOpen returns whether the handle is currently open.
func (w *WinDivert) IsOpen() bool {
	return w.open.Load()
}
