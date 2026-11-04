package main

import (
	"fmt"
	"math/rand/v2"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

const (
	width, height = 640, 360
	gravity       = 900
)

// Bunny is a sprite and its speed.
type Bunny struct {
	sprite *twod.Sprite
	speed  geom.Vec2
}

// Bunnies adds bunnies while Space or the mouse button is held, makes them
// fall and bounce, and writes how many there are.
type Bunnies struct {
	texture *twod.Texture
	window  *platform.Window
	field   *twod.Layer
	counter *twod.Text
	all     []*Bunny
	random  *rand.Rand
}

func newBunnies(texture *twod.Texture, window *platform.Window, field *twod.Layer, counter *twod.Text) *Bunnies {
	random := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a game's randomness, not a secret
	return &Bunnies{texture: texture, window: window, field: field, counter: counter, random: random}
}

func (b *Bunnies) Update(f world.Frame) {
	keys := b.window.Input()
	switch {
	case len(b.all) == 0:
		b.add(500, true)
	case keys.Down(input.KeySpace), keys.Down(input.MouseLeft):
		b.add(100, false)
	}

	for _, bunny := range b.all {
		bunny.speed.Y += gravity * f.DT
		bunny.sprite.Translate(bunny.speed.Scale(f.DT))
		at := bunny.sprite.Position()
		if at.X < 0 || at.X > width {
			bunny.speed.X = -bunny.speed.X
			at.X = geom.Clamp(at.X, 0, width)
		}
		if at.Y > height {
			bunny.speed.Y = -bunny.speed.Y * 0.85
			if b.random.Float32() < 0.5 {
				bunny.speed.Y -= 300 + 300*b.random.Float32()
			}
			at.Y = height
		}
		bunny.sprite.SetPosition(at)
	}

	b.counter.SetText(fmt.Sprintf("%d bunnies · %.0f fps", len(b.all), f.World.Stats().FPS))
}

// add puts n bunnies in the field, scattered over the window or at its
// top-left corner, each with its own speed. A child added to an entity
// already in the World is added to the World too.
func (b *Bunnies) add(n int, scattered bool) {
	for range n {
		at := geom.V2(0, 0)
		if scattered {
			at = geom.V2(width*b.random.Float32(), height*b.random.Float32())
		}
		bunny := &Bunny{
			sprite: twod.NewSprite(b.texture, twod.At(at.X, at.Y)),
			speed:  geom.V2(50+250*b.random.Float32(), 250*b.random.Float32()-125),
		}
		b.all = append(b.all, bunny)
		b.field.Add(bunny.sprite)
	}
}
