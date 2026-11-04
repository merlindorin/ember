//go:build windows || js

package collide //nolint:testpackage // white-box: cpu is shared with the white-box bench_test.go

import "time"

// cpu is the wall clock in nanoseconds: Windows and js have no getrusage.
func cpu() int64 { return time.Now().UnixNano() }
