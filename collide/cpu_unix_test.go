//go:build !windows && !js

package collide //nolint:testpackage // white-box: cpu is shared with the white-box bench_test.go

import "syscall"

// cpu is the process's CPU time in nanoseconds, steadier than the wall
// clock on a busy machine.
func cpu() int64 {
	var u syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &u)
	return u.Utime.Nano() + u.Stime.Nano()
}
