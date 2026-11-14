// Clipping: a layer that shows its items only inside a rectangle. The same
// news ticker scrolls twice: freely across the window at the top, and
// clipped to a framed strip below, where it slides in and out of view.
//
//	go run ./examples/25-clipping
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

const news = "Lighthouse keeper spots three ships at dusk · Harbour opens at dawn · Fog expected tonight"

// strip is where the clipped ticker shows, in window points.
func strip() geom.Rect {
	return geom.Rect{X: 100, Y: 200, W: 440, H: 64}
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · clipping"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	free := twod.NewText(news,
		twod.At(640, 80),
		twod.Pivot(0, 0.5),
		twod.Typeface(twod.Bold()),
		twod.FontSize(30),
	)
	clipped := twod.NewText(news,
		twod.At(640, strip().Y+strip().H/2),
		twod.Pivot(0, 0.5),
		twod.Typeface(twod.Bold()),
		twod.FontSize(30),
		twod.Color(color.Hex("#ffd166")),
	)

	top := twod.NewLayer(0)
	top.Add(free)
	ticker := twod.NewLayer(1, twod.ClipTo(strip()))
	ticker.Add(clipped)

	frame := twod.NewLine([]geom.Vec2{
		geom.V2(strip().X, strip().Y), geom.V2(strip().X+strip().W, strip().Y),
		geom.V2(strip().X+strip().W, strip().Y+strip().H), geom.V2(strip().X, strip().Y+strip().H),
	}, twod.Width(3), twod.Closed(), twod.Color(color.Hex("#6fb3d2")))

	win.Add(top, ticker, frame)

	ember.New().
		AddEntities(win).
		AddSystems(&Scroll{texts: []*twod.Text{free, clipped}}).
		MustRun()
}
