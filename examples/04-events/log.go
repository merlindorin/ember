package main

import (
	"fmt"
	"os"

	"github.com/lantervale/ember/world"
)

// Log reads every WentOut event once.
type Log struct{ wentOut world.Reader[WentOut] }

func (s *Log) Update(f world.Frame) {
	for event := range s.wentOut.Read(f.World) {
		fmt.Fprintf(os.Stdout, "%.1fs  %s went out\n", f.Elapsed, event.Name)
	}
}
