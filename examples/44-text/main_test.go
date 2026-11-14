package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheParagraphWrapsAndTheClockRuns(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	paragraph := twod.NewText(story, twod.At(24, 124), twod.Pivot(0, 0), twod.Wrap(300), twod.FontSize(18))
	clock := twod.NewText("")
	win.AppendChild(paragraph)
	win.AppendChild(clock)
	w := ember.New().AddEntities(win).AddSystems(&Clock{text: clock})
	w.RunFrames(60)

	if lines := paragraph.Lines(); len(lines) < 3 {
		t.Fatalf("the story wraps at 300 points: %q", lines)
	}
	if clock.Text() != "001.0 s" {
		t.Fatalf("sixty frames are one second: %q", clock.Text())
	}
}
