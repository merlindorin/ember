package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// Loading fills the bar in four seconds, and starts again.
type Loading struct {
	doc   *ui.Document
	bar   *ui.Node
	value float64
}

func (l *Loading) Update(f world.Frame) {
	l.value += float64(f.DT) / 4
	if l.value > 1 {
		l.value = 0
	}
	kit.SetProgress(l.bar, l.value)
}
