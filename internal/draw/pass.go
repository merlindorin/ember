package draw

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
)

// Pass is an offscreen picture its window draws before itself: List, in
// Screen points, over a transparent image of Width×Height pixels that
// Target names, then through Effects in order.
type Pass struct {
	// Name says what draws it, a layer's or a camera's name, for profilers.
	Name          string
	Target        *render.Texture
	Width, Height int
	Screen        geom.Vec2
	List          *List
	Effects       []*render.Effect
}

// NewPass returns a pass with its own target and list.
func NewPass() *Pass { return &Pass{Target: &render.Texture{}, List: &List{}} }

// Picture draws the whole of p over r, blended by m and through mask's
// alpha when there is one.
func (c *List) Picture(r geom.Rect, p *Pass, m Blend, mask *Pass) {
	c.SetTransform(geom.Identity2D())
	c.SetModulate(color.White)
	light := c.light
	c.light = Unlit
	c.vertices(r, [4]float32{0, 0, 1, 1}, color.White, geom.Vec4{X: KindImage})
	c.light = light
	b := Batch{First: len(c.Indices) - 6, Count: 6, Blend: m, alone: true}
	b.Images[0], b.ImageCount = p.Target, 1
	b.Clip, b.Clipped = c.clip()
	b.Round = c.round()
	if mask != nil {
		b.Mask = mask.Target
	}
	c.Batches = append(c.Batches, b)
}

// Source is what a World's windows drew: twod installs one, presenters and
// engine tests read it.
type Source interface {
	// Front is what shows over the 3D scene, Back what shows behind it, and
	// Passes the pictures drawn first, in order.
	Front(win *platform.Window) *List
	Back(win *platform.Window) *List
	Passes(win *platform.Window) []*Pass
	// Classes are win's light classes this frame, from 1: for each, the
	// layers some light Affects that it is under, the innermost first.
	Classes(win *platform.Window) [][]any
	// ToScreen is where e, a 2D entity, showed in its window in the last
	// frame, in points.
	ToScreen(e any) (geom.Vec2, bool)
}

type source struct{ Source }

// Install makes s the World's source of drawing.
func Install(w *world.World, s Source) { slot.Set(w, &source{s}) }

func of(w *world.World) Source {
	if s := slot.Get[source](w); s != nil {
		return s.Source
	}
	return nil
}

// FrontFor is what win shows over the 3D scene, nil when nothing drew.
func FrontFor(w *world.World, win *platform.Window) *List {
	if s := of(w); s != nil {
		return s.Front(win)
	}
	return nil
}

// ClassesFor is win's light classes this frame, from class 1 (see
// Source.Classes).
func ClassesFor(w *world.World, win *platform.Window) [][]any {
	if s := of(w); s != nil {
		return s.Classes(win)
	}
	return nil
}

// BackFor is what win shows behind the 3D scene, nil when nothing drew.
func BackFor(w *world.World, win *platform.Window) *List {
	if s := of(w); s != nil {
		return s.Back(win)
	}
	return nil
}

// ToScreen is where e, a 2D entity, showed in its window in the last
// frame, in points.
func ToScreen(w *world.World, e any) (geom.Vec2, bool) {
	if s := of(w); s != nil {
		return s.ToScreen(e)
	}
	return geom.Vec2{}, false
}

// PassesFor are the passes win draws this frame, in order.
func PassesFor(w *world.World, win *platform.Window) []*Pass {
	if s := of(w); s != nil {
		return s.Passes(win)
	}
	return nil
}
