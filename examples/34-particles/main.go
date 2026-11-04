// Particles: a campfire. Flames and sparks add their light (Blend(Add)),
// smoke drifts up and fades over them, and a click throws a one-shot
// burst of embers that falls under gravity and removes itself when done.
//
//	go run ./examples/34-particles
package main

import (
	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func main() {
	win := platform.NewWindow(
		platform.Title("ember · particles"),
		platform.Size(480, 320),
		platform.FixedSize(),
		platform.Background(color.Hex("#0b1020")),
	)

	puff := glow(64)

	smoke := twod.NewParticles(puff,
		twod.Rate(16),
		twod.Lifetime(2.5, 4),
		twod.Velocity(25, 45),
		twod.Direction(-geom.Pi/2, 0.25),
		twod.Gravity(geom.V2(6, -4)),
		twod.ColorOverLife(color.RGBA(0.62, 0.62, 0.68, 0.75), color.RGBA(0.4, 0.4, 0.45, 0)),
		twod.SizeOverLife(18, 90),
		twod.Spin(-0.6, 0.6),
	)
	flames := twod.NewParticles(puff,
		twod.Rate(90),
		twod.Lifetime(0.5, 0.9),
		twod.Velocity(30, 70),
		twod.Direction(-geom.Pi/2, 0.45),
		twod.Gravity(geom.V2(0, -60)),
		twod.ColorOverLife(color.Hex("#ffcf6b"), color.RGBA(0.9, 0.15, 0.02, 0)),
		twod.SizeOverLife(80, 16),
		twod.Blend(twod.Add),
	)
	sparks := twod.NewParticles(nil,
		twod.Rate(18),
		twod.Lifetime(0.8, 1.6),
		twod.Velocity(90, 170),
		twod.Direction(-geom.Pi/2, 0.5),
		twod.Gravity(geom.V2(0, 60)),
		twod.ColorOverLife(color.Hex("#fff0a0"), color.RGBA(1, 0.4, 0.1, 0)),
		twod.SizeOverLife(3, 1),
		twod.Blend(twod.Add),
	)
	logs := twod.NewPolygon(
		[]geom.Vec2{geom.V2(-40, 0), geom.V2(40, -12), geom.V2(44, 0), geom.V2(-36, 12)},
		twod.Color(color.Hex("#5a3a22")),
	)
	fire := twod.NewLayer(0).Add(smoke, logs, flames, sparks)
	fire.SetPosition(geom.V2(240, 250))

	ground := twod.NewPolygon(
		[]geom.Vec2{geom.V2(0, 260), geom.V2(480, 260), geom.V2(480, 320), geom.V2(0, 320)},
		twod.Color(color.Hex("#1a2a1c")),
	)
	bursts := twod.NewLayer(1)
	hint := twod.NewLayer(2, twod.Fixed()).Add(
		twod.NewText("click to throw embers", twod.At(16, 12), twod.Pivot(0, 0)),
	)
	win.Add(ground, fire, bursts, hint)

	ember.New().
		AddEntities(win).
		AddSystems(&Bursts{window: win, layer: bursts}).
		MustRun()
}
