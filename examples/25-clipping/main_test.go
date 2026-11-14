package main

import (
	"testing"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func TestTheClippedTickerStaysInsideItsStrip(t *testing.T) {
	win := platform.NewWindow(platform.Size(640, 360))
	clipped := twod.NewText(news, twod.At(300, strip().Y+strip().H/2), twod.Pivot(0, 0.5))
	ticker := twod.NewLayer(1, twod.ClipTo(strip()))
	ticker.Add(clipped)
	win.AppendChild(ticker)
	w := ember.New().AddEntities(win)
	w.RunFrames(2)

	list := draw.FrontFor(w, win)
	if len(list.Batches) == 0 {
		t.Fatal("the ticker draws")
	}
	for _, b := range list.Batches {
		if !b.Clipped || b.Clip != strip() {
			t.Fatalf("every batch of the ticker is clipped to the strip: %+v", b)
		}
	}
	if clipped.Measure().X <= strip().W {
		t.Fatal("the news is longer than the strip, so clipping shows")
	}
}

func TestTickersComeBackFromTheRight(t *testing.T) {
	text := twod.NewText("short", twod.At(-200, 70), twod.Pivot(0, 0.5))
	w := ember.New().AddEntities(text).AddSystems(&Scroll{texts: []*twod.Text{text}})
	w.RunFrames(1)
	if text.Position().X != 640 {
		t.Fatalf("gone by, it starts again at the right edge: %v", text.Position())
	}
}
