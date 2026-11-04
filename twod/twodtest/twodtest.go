// Package twodtest reads what a window's 2D drawing and light hold after a
// frame, for a game's tests: where an entity was drawn, the pictures its
// offscreen passes make and the images the window shows, and the light it
// is lit by, with the shapes that cast its shadows. It is for tests only,
// never game code, and follows the renderer, so it may change with it.
package twodtest

import (
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// Drawn is where e was drawn in the last frame: its transform to its
// window's points (the primary window's for one in none), between fixed
// steps as the frame drew it. ok is false when the World draws no 2D.
func Drawn(w *world.World, e twod.Entity) (geom.Transform2D, bool) {
	t, _, ok := lit.Screen(w, e)
	return t, ok
}

// Pass is an offscreen pass of a window: what draws it (a layer's or a
// camera's name) and the picture it makes.
type Pass struct {
	Name   string
	Target *render.Texture
}

// Passes are the window's offscreen passes this frame, in the order the
// GPU draws them.
func Passes(w *world.World, win *platform.Window) []Pass {
	var out []Pass
	for _, p := range draw.PassesFor(w, win) {
		out = append(out, Pass{Name: p.Name, Target: p.Target})
	}
	return out
}

// Shown are the images the window's own drawing shows this frame, each
// once, passes' pictures among them.
func Shown(w *world.World, win *platform.Window) []*render.Texture {
	l := draw.FrontFor(w, win)
	if l == nil {
		return nil
	}
	return images(l)
}

func images(l *draw.List) []*render.Texture {
	var out []*render.Texture
	for _, b := range l.Batches {
		for _, img := range b.Images[:b.ImageCount] {
			if img != nil && !slices.Contains(out, img) {
				out = append(out, img)
			}
		}
	}
	return out
}

// Lighting is the light a window is lit by this frame, in its points.
type Lighting struct {
	Ambient color.Color
	Lights  []Light
	// Occluders are the vertices of the shapes that cast shadows, in the
	// order the GPU draws them (four to a box), each with its occluder
	// groups as bits: 1<<g for group g.
	Occluders []OccluderVertex
	// AmbientMap is the ambient picture's tiles in view.
	AmbientMap []*render.Texture
}

// Light is one light: where it stands, how far it reaches and falls off,
// its colour, its height and its size for its shadows.
type Light struct {
	At                    geom.Vec2
	Radius, Falloff       float32
	Color                 color.Color
	Elevation, ShadowSize float32
}

// OccluderVertex is a vertex of a shadow-casting shape and its groups.
type OccluderVertex struct {
	At     geom.Vec2
	Groups uint8
}

// LightingOf is the light win is lit by this frame; ok is false when
// nothing lights it.
func LightingOf(w *world.World, win *platform.Window) (Lighting, bool) {
	l := draw.LightingFor(w, win)
	if l == nil {
		return Lighting{}, false
	}
	// Occluders are drawn from the light buffer's corner, past the window's.
	corner := draw.LightRect(win.Size())
	at := func(p geom.Vec2) geom.Vec2 { return p.Add(geom.V2(corner.X, corner.Y)) }
	out := Lighting{Ambient: l.Ambient}
	for _, li := range l.Lights {
		out.Lights = append(
			out.Lights,
			Light{
				At:         li.At,
				Radius:     li.Radius,
				Falloff:    li.Falloff,
				Color:      li.Color,
				Elevation:  li.Elevation,
				ShadowSize: li.ShadowSize,
			},
		)
	}
	if l.Occluders != nil {
		for _, v := range l.Occluders.Vertices {
			out.Occluders = append(out.Occluders, OccluderVertex{At: at(v.Pos), Groups: groups(v.Color)})
		}
	}
	if l.AmbientMap != nil {
		out.AmbientMap = images(l.AmbientMap)
	}
	return out, true
}

// groups reads an occluder's groups back from its colour (draw.GroupColor).
func groups(c color.Color) uint8 {
	var g uint8
	for i, on := range []float32{c.R, c.G, c.B} {
		if on > 0.5 {
			g |= 1 << i
		}
	}
	return g
}

// AnswerProbes makes every light probe asked in win this frame read light,
// as if the GPU had answered: for tests without one.
func AnswerProbes(w *world.World, win *platform.Window, light color.Color) {
	l := draw.LightingFor(w, win)
	if l == nil {
		return
	}
	for _, p := range l.Probes {
		*p.Result = draw.ProbeResult{Light: light, Ready: true}
	}
}
