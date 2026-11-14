package twod

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/node"
)

// BlendMode is how an entity's colours combine with what is under them.
type BlendMode int

const (
	// Alpha covers what is under by the entity's opacity.
	Alpha BlendMode = iota
	// Add lights what is under up: glows, fire, sparks.
	Add
	// Multiply darkens what is under: shadows, night.
	Multiply
	// Screen lightens what is under softly: haze, light shafts.
	Screen
	// Lighten keeps, per channel, the brighter of the entity and what is
	// under it: an ambient light under which lights add up.
	Lighten
)

//nolint:gochecknoglobals // word table
var blendWords = node.Words[BlendMode]{"alpha", "add", "multiply", "screen", "lighten"}

func (m BlendMode) MarshalText() ([]byte, error)  { return blendWords.Text(m) }
func (m *BlendMode) UnmarshalText(b []byte) error { return blendWords.Parse(m, b) }

// Blend sets how the entity itself, not its children, combines with what
// is under it. A layer's Blend combines the layer's whole picture.
func Blend(m BlendMode) Option { return func(e Entity) { e.GetDrawLook().blend = m } }

// AsMask makes a layer a mask: it is never shown, and the layers MaskedBy
// it show only where it is opaque.
func AsMask() LayerOption { return layerOption(func(l *Layer) { l.mask = true }) }

// MaskedBy shows a layer only where mask, a layer made with AsMask, is
// opaque: a spotlight, a fog of war.
func MaskedBy(mask *Layer) LayerOption { return layerOption(func(l *Layer) { l.maskedBy = mask }) }

// pictured reports whether the layer draws into a pass of its own, shown
// as one picture: to blend it as a whole, for a mask, or for its effects.
func (l *Layer) pictured() bool {
	return l.blend != Alpha || l.mask || l.maskedBy != nil || len(l.effects) > 0
}

// picturePass is the layer's pass, sized to a surface of screen points
// and fb pixels.
func (l *Layer) picturePass(screen, fb geom.Vec2) *draw.Pass {
	if l.pass == nil {
		l.pass = draw.NewPass()
	}
	return l.sized(l.pass, screen, fb)
}

// sized sets p up as l's picture over a surface of screen points and fb
// pixels.
func (l *Layer) sized(p *draw.Pass, screen, fb geom.Vec2) *draw.Pass {
	p.Width, p.Height, p.Screen = max(int(fb.X*l.passScale+0.5), 1), max(int(fb.Y*l.passScale+0.5), 1), screen
	p.Effects = l.effects
	p.Target.Filter = Linear
	p.Name = l.GetSceneNode().Name()
	return p
}

// pictureIn is l's pass in cam's render texture, apart from the one the
// window draws it through.
func (l *Layer) pictureIn(cam *Camera, size geom.Vec2) *draw.Pass {
	if cam.pictures == nil {
		cam.pictures = map[*Layer]*draw.Pass{}
	}
	p, ok := cam.pictures[l]
	if !ok {
		p = draw.NewPass()
		cam.pictures[l] = p
	}
	return l.sized(p, size, size)
}
