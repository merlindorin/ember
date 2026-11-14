// Lines and shapes: three zigzags show how a line turns at its points
// (miter, bevel, round) and how it ends (flat, square, round). A closed line
// outlines a star, and a polygon fills a concave arrow.
//
//	go run ./examples/21-lines-and-shapes
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
)

func zigzag() []geom.Vec2 {
	return []geom.Vec2{geom.V2(0, 40), geom.V2(50, 0), geom.V2(100, 40), geom.V2(150, 0)}
}

func star() []geom.Vec2 {
	return []geom.Vec2{
		geom.V2(0, -60), geom.V2(18, -20), geom.V2(60, -20), geom.V2(26, 6), geom.V2(38, 48),
		geom.V2(0, 22), geom.V2(-38, 48), geom.V2(-26, 6), geom.V2(-60, -20), geom.V2(-18, -20),
	}
}

func arrow() []geom.Vec2 {
	return []geom.Vec2{
		geom.V2(0, 30), geom.V2(70, 30), geom.V2(70, 0), geom.V2(130, 50),
		geom.V2(70, 100), geom.V2(70, 70), geom.V2(0, 70),
	}
}

func main() {
	win := platform.NewWindow(
		platform.Title("ember · lines and shapes"),
		platform.Size(640, 360),
		platform.FixedSize(),
	)

	miter := twod.NewLine(zigzag(),
		twod.At(40, 50),
		twod.Width(14),
		twod.Joints(twod.Miter),
		twod.Caps(twod.Flat, twod.Flat),
		twod.Color(color.Hex("#e8a33d")),
	)
	bevel := twod.NewLine(zigzag(),
		twod.At(240, 50),
		twod.Width(14),
		twod.Joints(twod.Bevel),
		twod.Caps(twod.Square, twod.Square),
		twod.Color(color.Hex("#6fb3d2")),
	)
	round := twod.NewLine(zigzag(),
		twod.At(440, 50),
		twod.Width(14),
		twod.Joints(twod.RoundJoint),
		twod.Caps(twod.RoundCap, twod.RoundCap),
		twod.Color(color.Hex("#9ccc65")),
	)
	outline := twod.NewLine(star(),
		twod.At(170, 250),
		twod.Width(4),
		twod.Closed(),
		twod.Color(color.Hex("#f2e6c9")),
	)
	filled := twod.NewPolygon(arrow(),
		twod.At(370, 200),
		twod.Color(color.Hex("#d9534f")),
	)
	win.Add(miter, bevel, round, outline, filled)

	ember.New().
		AddEntities(win).
		MustRun()
}
