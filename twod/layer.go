package twod

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/render"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewLayer(0) }})
}

// Layer groups 2D entities drawn together: layers draw in number order,
// the lowest first. A Fixed layer stays on screen when the camera moves,
// for a HUD; a Parallax one moves slower or faster than the camera.
type Layer struct {
	Item
	number    int       `node:"prop"`
	fixed     bool      `node:"prop"`
	parallax  float32   `node:"prop"`
	clip      geom.Rect `node:"prop"`
	anchor    geom.Vec2 `node:"prop"`
	mask      bool      `node:"prop"`
	maskedBy  *Layer
	effects   []*render.Effect
	passScale float32 `node:"prop"`
	pass      *draw.Pass
	// grid files its children by where they are, once it has many;
	// gridIdle counts the frames left before it tries again.
	grid     *grid
	gridIdle int
}

// NewLayer returns layer number n.
func NewLayer(n int, opts ...LayerOption) *Layer {
	l := Build(&Layer{Item: Base(), number: n, parallax: 1, passScale: 1}, "layer")
	for _, o := range opts {
		o.layer(l)
	}
	return l
}

// Fixed keeps a layer on screen whatever the camera does.
func Fixed() LayerOption { return layerOption(func(l *Layer) { l.fixed = true }) }

// Anchor places a Fixed layer's origin at x, y of its window, as
// fractions: 1, 0 is the top-right corner, 0.5, 1 the middle of the bottom
// edge. Its entities then keep their place from that edge as the window
// resizes. The default is 0, 0, the top-left corner.
func Anchor(x, y float32) LayerOption {
	return layerOption(func(l *Layer) { l.anchor = geom.V2(x, y) })
}

// PassScale draws a layer that shows as one picture (a Blend, a mask,
// an effect) at f times its window's resolution, from 0 to 1, stretched
// smooth over the window: a light map or a fog, soft anyway, filled at a
// fraction of the cost. Other layers ignore it.
func PassScale(f float32) LayerOption { return layerOption(func(l *Layer) { l.SetPassScale(f) }) }

// Parallax moves a layer at f times the camera's speed: 0.5 for a far
// background.
func Parallax(f float32) LayerOption { return layerOption(func(l *Layer) { l.parallax = f }) }

// ClipTo shows only what of a layer falls inside r, in the layer's space.
func ClipTo(r geom.Rect) LayerOption { return layerOption(func(l *Layer) { l.clip = r }) }

// WithEffect draws a layer through e, a render.NewEffect: a WGSL effect
// function with the inputs of render.Effect and no depth. color(uv) reads
// the layer's pixels, their colours multiplied by their alpha, and the
// effect returns them the same way, with vUV.y growing down the screen.
// Given more than once, the effects run in order. A shader that fails to
// compile is skipped, and e.Err says why.
func WithEffect(e *render.Effect) LayerOption {
	return layerOption(func(l *Layer) {
		if e != nil {
			l.effects = append(l.effects, e)
		}
	})
}

func (l *Layer) Number() int { return l.number }
func (l *Layer) Fixed() bool { return l.fixed }
