package desktop

import (
	"syscall"
	"unsafe"
)

// showError shows a native message box with message, titled title, so a
// windowed build never fails without a word.
func showError(title, message string) {
	box := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	t, err1 := syscall.UTF16PtrFromString(title)
	m, err2 := syscall.UTF16PtrFromString(message)
	if err1 != nil || err2 != nil {
		return
	}
	const iconError = 0x10
	_, _, _ = box.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), iconError) // nothing to fall back to
}
