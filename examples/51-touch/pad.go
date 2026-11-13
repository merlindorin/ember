package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// pad is where the trackpad is drawn, its shape a trackpad's.
func pad() geom.Rect {
	return geom.Rect{X: 80, Y: 50, W: 480, H: 300}
}

// onPad is the point of the drawn pad where a touch at at, 0 to 1 on the
// trackpad, is.
func onPad(at geom.Vec2) geom.Vec2 {
	return geom.V2(pad().X+at.X*pad().W, pad().Y+at.Y*pad().H)
}

func newPad() *twod.Polygon {
	return twod.NewPolygon([]geom.Vec2{
		{X: pad().X, Y: pad().Y}, {X: pad().X + pad().W, Y: pad().Y},
		{X: pad().X + pad().W, Y: pad().Y + pad().H}, {X: pad().X, Y: pad().Y + pad().H},
	}, twod.Color(color.Hex("#2c323d")))
}

// disc is the points of a circle of radius r around the origin.
func disc(r float32) []geom.Vec2 {
	points := make([]geom.Vec2, 32)
	for i := range points {
		a := float32(i) / float32(len(points)) * 2 * geom.Pi
		points[i] = geom.V2(r*geom.Cos(a), r*geom.Sin(a))
	}
	return points
}
