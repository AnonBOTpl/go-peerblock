//go:build windivert

package filter

/*
#cgo LDFLAGS: -L${SRCDIR}/.. -lWinDivert
#define WINDIVERTEXPORT
#include <windivert.h>
*/
import "C"
import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

// WinDivert wraps the native WinDivert handle.
type WinDivert struct {
	handle C.HANDLE
	filter string
	open   atomic.Bool
}

// Open creates a new WinDivert handle.
func Open(filter string, layer int32, priority int16) (*WinDivert, error) {
	cFilter := C.CString(filter)
	defer C.free(unsafe.Pointer(cFilter))

	h := C.WinDivertOpen(cFilter, C.WINDIVERT_LAYER(layer), C.INT16(priority), 0)
	if uintptr(unsafe.Pointer(h)) == uintptr(^uintptr(0)) {
		return nil, fmt.Errorf("WinDivertOpen failed (filter=%q)", filter)
	}

	w := &WinDivert{
		handle: h,
		filter: filter,
	}
	w.open.Store(true)
	return w, nil
}

// Recv receives a packet and returns its data, address, and error.
func (w *WinDivert) Recv(buf []byte) (int, interface{}, error) {
	var addr C.WINDIVERT_ADDRESS
	var readLen C.UINT

	ret := C.WinDivertRecv(
		w.handle,
		unsafe.Pointer(&buf[0]),
		C.UINT(len(buf)),
		&readLen,
		&addr,
	)
	if ret == 0 {
		return 0, nil, fmt.Errorf("WinDivertRecv failed")
	}

	// Copy address into Go heap so it survives beyond this call
	addrCopy := make([]byte, C.sizeof_WINDIVERT_ADDRESS)
	addrPtr := (*C.WINDIVERT_ADDRESS)(unsafe.Pointer(&addrCopy[0]))
	*addrPtr = addr

	return int(readLen), addrCopy, nil
}

// Send sends a packet back to the network stack with the original address.
func (w *WinDivert) Send(buf []byte, addr interface{}) (int, error) {
	var addrPtr *C.WINDIVERT_ADDRESS
	if addr != nil {
		addrBytes, ok := addr.([]byte)
		if ok && len(addrBytes) >= int(C.sizeof_WINDIVERT_ADDRESS) {
			addrPtr = (*C.WINDIVERT_ADDRESS)(unsafe.Pointer(&addrBytes[0]))
		}
	}

	var sendLen C.UINT
	ret := C.WinDivertSend(
		w.handle,
		unsafe.Pointer(&buf[0]),
		C.UINT(len(buf)),
		&sendLen,
		addrPtr,
	)
	if ret == 0 {
		return 0, fmt.Errorf("WinDivertSend failed")
	}
	return int(sendLen), nil
}

// Close closes the WinDivert handle.
func (w *WinDivert) Close() error {
	w.open.Store(false)
	if C.WinDivertClose(w.handle) == 0 {
		return fmt.Errorf("WinDivertClose failed")
	}
	return nil
}

// IsOpen returns whether the handle is currently open.
func (w *WinDivert) IsOpen() bool {
	return w.open.Load()
}
