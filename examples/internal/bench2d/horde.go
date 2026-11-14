package main

import (
	"math/rand/v2"

	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/examples/internal/crypt"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/world"
)

// foe is a demo enemy as the demo builds one: an entity with a sprite and
// a collide body as children, three World entities in all.
type foe struct {
	twod.Item
	sprite *twod.Sprite
	walk   geom.Vec2
	speed  float32
}

// chase points every foe at the hero once a frame, before the fixed steps,
// as the demo's Chase does.
type chase struct {
	world.Group[*foe]
	hero *twod.Sprite
}

func (s *chase) Phase() world.Phase { return world.PreUpdate }

func (s *chase) Update(world.Frame) {
	at := s.hero.Position()
	for e := range s.All() {
		e.walk = at.Sub(e.Position()).Normalize()
	}
}

// step moves every foe through the collision space in fixed steps, as the
// demo's Move does, the soft crowd spreading out.
type step struct {
	world.Group[*foe]
	space *collide.Space
}

func (s *step) Phase() world.Phase { return world.FixedUpdate }

func (s *step) Update(f world.Frame) {
	for e := range s.All() {
		s.space.Move(e, e.walk.Scale(e.speed*f.DT))
	}
}

// look plays each foe's run and faces it the way it walks, once a frame,
// as the demo's Look does.
type look struct{ world.Group[*foe] }

func (s *look) Update(world.Frame) {
	for e := range s.All() {
		e.sprite.Play("run")
		if e.walk.X != 0 {
			e.sprite.SetFlip(e.walk.X < 0, false)
		}
	}
}

// horde adds n foes of the crypt's four breeds to crowd, spread evenly over
// a disc of radius around the origin, and the systems that move them as
// the demo does: chasing hero, a soft crowd spreading in a grid of 32.
func horde(w *world.World, crowd *twod.Layer, hero *twod.Sprite, n int, radius float32, r *rand.Rand) {
	space := collide.NewSpace(collide.Grid(32))
	w.AddSystems(collide.NewTracker(space), &chase{hero: hero}, &step{space: space}, &look{})
	const foes = 2
	for i, breed := range []string{"goblin", "skelet", "imp", "orc_warrior"} {
		sheet, clips := crypt.Character(breed)
		for range n / 4 {
			at := geom.FromAngle(r.Float32() * 2 * geom.Pi).Scale(radius * geom.Sqrt(r.Float32()))
			sprite := twod.NewSprite(
				sheet,
				twod.Animate(clips),
				twod.Size(32+float32(i/3)*4, 32+float32(i/3)*14),
				twod.Pivot(0.5, 1),
			)
			e := twod.Build(
				&foe{Item: twod.Base(), sprite: sprite, speed: 40 + r.Float32()*45},
				breed,
				twod.At(at.X, at.Y),
			)
			e.Add(
				sprite,
				collide.NewBody(
					collide.Box(16, 10),
					collide.Offset(0, -5),
					collide.Group(foes),
					collide.CollidesWith(foes),
					collide.Soft(1),
				),
			)
			crowd.Add(e)
		}
	}
}
