package main

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// loop is the points a closed path goes through, back to the first.
func loop() []geom.Vec2 {
	return []geom.Vec2{
		geom.V2(560, 120), geom.V2(680, 90), geom.V2(720, 220), geom.V2(640, 300),
		geom.V2(700, 420), geom.V2(560, 440), geom.V2(520, 300),
	}
}

// newPath draws the closed Catmull-Rom curve through loop and sends a
// walker round it every six seconds.
func newPath(walker *twod.Texture) *twod.Layer {
	var curve []geom.Vec2
	for i := range 121 {
		curve = append(curve, geom.ClosedCatmullRom(loop(), float32(i)/120))
	}
	runner := twod.NewSprite(walker, twod.Still(twod.Strip(walker, twod.Cell(16, 16))[0]), twod.Scale(3, 3))
	runner.Add(ease.NewTween(6, func(t float32) {
		runner.SetPosition(geom.ClosedCatmullRom(loop(), t))
	}, ease.Loop()))
	return twod.NewLayer(0).Add(
		twod.NewLine(curve, twod.Width(2), twod.Color(color.Hex("#4a5578"))),
		runner,
	)
}
