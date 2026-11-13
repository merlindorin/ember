package main

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The corridor and the hall are the floors the hero walks on.
func corridorFloor() geom.Rect { return geom.Rect{X: 0, Y: 150, W: 270, H: 60} }
func hallFloor() geom.Rect     { return geom.Rect{X: 270, Y: 70, W: 350, H: 260} }

func box(r geom.Rect) []geom.Vec2 {
	return []geom.Vec2{{X: r.X, Y: r.Y}, {X: r.X + r.W, Y: r.Y}, {X: r.X + r.W, Y: r.Y + r.H}, {X: r.X, Y: r.Y + r.H}}
}

func newDungeon() *twod.Item {
	wall, floor := color.Hex("#4a4458"), color.Hex("#6b6478")
	d := twod.Base()
	return d.Add(
		twod.NewPolygon(box(corridorFloor().Inset(-8)), twod.Color(wall)),
		twod.NewPolygon(box(hallFloor().Inset(-8)), twod.Color(wall)),
		twod.NewPolygon(box(corridorFloor()), twod.Color(floor)),
		twod.NewPolygon(box(hallFloor()), twod.Color(floor)),
		twod.NewText("corridor", twod.At(110, 222), twod.Pivot(0.5, 0), twod.FontSize(13)),
		twod.NewText("great hall", twod.At(445, 300), twod.Pivot(0.5, 0), twod.FontSize(13)),
	)
}

// onFloor reports whether every corner of the hero's feet, standing at at,
// is on the corridor's or the hall's floor.
func onFloor(at geom.Vec2) bool {
	for _, c := range []geom.Vec2{{X: -8, Y: -6}, {X: 8, Y: -6}, {X: -8}, {X: 8}} {
		p := at.Add(c)
		if !corridorFloor().Contains(p) && !hallFloor().Contains(p) {
			return false
		}
	}
	return true
}
