//go:build chrome && windows

package chrome

import (
	"os"

	"golang.org/x/sys/windows"
)

// lockedByte is the one byte the lock covers, far past the holder's note:
// Windows locks are mandatory, and those waiting still read whom for.
const lockedByte = 1 << 40

// lockFile takes f's exclusive lock, waiting for it when wait is set, else
// failing at once while another holds it.
func lockFile(f *os.File, wait bool) error {
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK)
	if !wait {
		flags |= windows.LOCKFILE_FAIL_IMMEDIATELY
	}
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, at())
}

func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, at())
}

func at() *windows.Overlapped {
	return &windows.Overlapped{Offset: lockedByte & 0xffffffff, OffsetHigh: lockedByte >> 32}
}
