// Command large opens a window larger than any CI screen, for the golden
// package's test that a hidden capture keeps the size the window asked.
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
)

func main() {
	ember.New().AddEntities(platform.NewWindow(platform.Title("Golden Large"), platform.Size(2400, 1400))).MustRun()
}
