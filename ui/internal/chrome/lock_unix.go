//go:build chrome && !windows

package chrome

import (
	"os"
	"syscall"
)

// lockFile takes f's exclusive lock, waiting for it when wait is set, else
// failing at once while another holds it.
func lockFile(f *os.File, wait bool) error {
	how := syscall.LOCK_EX
	if !wait {
		how |= syscall.LOCK_NB
	}
	return syscall.Flock(int(f.Fd()), how)
}

func unlockFile(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
