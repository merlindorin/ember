// Package light lights 2D layers on the GPU: lights, the shadows what
// blocks them casts, and the ambient light every spot gets anyway
// (decision 0042, docs/design/2d-lighting.md).
//
// A layer made with Lit() multiplies its sprites by the light falling on
// them; Unlit() leaves a sprite as it is, and LitAt(x, y) lights a whole
// sprite as one point of it. Lights are 2D entities, children of what they
// follow: points, cones, directional lights across the whole view, and
// shapes. Collide solids and hard bodies (Occluders on the camera) and
// sprites made with Occluder() block light; Emissive sprites glow. A game
// reads the light where something stands through a Probe, answered by the
// GPU a few frames later.
//
// Limits: a window draws 256 lights, of which 256 shadow rows (a shape
// light with shadows takes 8), and 4 directional lights; occluders come in
// 3 groups. A camera drawing Into a render texture draws lit layers unlit.
// The ambient is one colour (Ambient), or a tiled picture the game paints
// (AmbientMap).
package light

import (
	"slices"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/twod"
)

// Light is a light in the 2D world.
type Light struct {
	twod.Item
	kind      draw.LightKind
	radius    float32
	color     color.Color
	falloff   float32
	intensity float32
	xray      bool
	groups    uint8
	angle     float32
	spread    float32
	length    float32
	elevation float32
	size      float32
	fade      Fade
	fadeFrom  float32
	shape     Shape
	static    bool
	priority  int
	// shadowColor and shadowOpacity are what its shadows let through.
	shadowColor   color.Color
	shadowOpacity float32
	// affects are the layers it lights, nil for every one.
	affects []*twod.Layer
}

// Option sets up a light.
type Option func(*Light)

func newLight(kind draw.LightKind, radius float32, c color.Color, opts []Option) *Light {
	l := twod.Build(&Light{Item: twod.Base(), kind: kind, radius: radius, color: c, falloff: 0.25, intensity: 1,
		groups: allGroups, length: 256, shadowOpacity: 1}, "light")
	if kind == draw.ShapeLight {
		l.falloff, l.xray = 0, true
	}
	for _, o := range opts {
		o(l)
	}
	return l
}

const allGroups = 1<<draw.OccluderGroups - 1

// NewPoint returns a light of radius points in colour c: a soft disc round
// where it stands, full out to its falloff, then fading.
func NewPoint(radius float32, c color.Color, opts ...Option) *Light {
	return newLight(draw.PointLight, radius, c, opts)
}

// NewCone returns a point light shining only along angle, in radians in
// its own space (0 along x, turning with what it follows), spread radians
// wide: a flashlight.
func NewCone(radius float32, c color.Color, angle, spread float32, opts ...Option) *Light {
	l := newLight(draw.ConeLight, radius, c, opts)
	l.angle, l.spread = angle, spread
	return l
}

// NewDirectional returns a light across the whole view, its rays going
// along angle, in radians: the sun or the moon. Its shadows are
// ShadowLength points long.
func NewDirectional(c color.Color, angle float32, opts ...Option) *Light {
	l := newLight(draw.DirectionalLight, 0, c, opts)
	l.angle = angle
	return l
}

// NewShape returns a light full inside shape, in its own space, fading to
// nothing reach points past it: a window, a lava pool, a neon tube. It
// casts no shadow unless given Shadows.
func NewShape(shape Shape, c color.Color, reach float32, opts ...Option) *Light {
	l := newLight(draw.ShapeLight, reach, c, opts)
	l.shape = shape
	return l
}

// At places the light, in its parent's space.
func At(x, y float32) Option { return func(l *Light) { l.SetPosition(geom.V2(x, y)) } }

// Falloff is the share of the radius at full strength: 0 fades from the
// centre, 1 is a hard edge; 0.25 by default. Past it the light fades to
// nothing at the radius. A shape light fades over all its reach unless
// given one.
func Falloff(full float32) Option { return func(l *Light) { l.falloff = clamp01(full) } }

// Intensity scales the light's colour, 1 by default.
func Intensity(i float32) Option { return func(l *Light) { l.intensity = max(i, 0) } }

// Shadows says whether the light casts shadows: point, cone and
// directional lights do by default, shining through nothing; a shape
// light doesn't, and casts them from points along its outline when on.
// Off, a light shines through what blocks light, as an x-ray would.
func Shadows(on bool) Option { return func(l *Light) { l.xray = !on } }

// BlockedBy makes only the occluders of groups (0, 1 or 2) cast the
// light's shadows; by default all do.
func BlockedBy(groups ...int) Option { return func(l *Light) { l.groups = groupBits(groups) } }

// Elevation is how high the light stands, in points: an occluder of height
// h, at d from it, casts a shadow ending at d·e/(e−h), so a low lamp casts
// long shadows and a high one short ones; one as tall as the lamp or
// taller, or of no height given, casts to the radius. 0, the default,
// casts every shadow to the radius.
func Elevation(points float32) Option { return func(l *Light) { l.elevation = max(points, 0) } }

// Affects makes the light light only these layers, and every layer and
// entity they hold, nested ones too; by default it lights every lit
// layer. Each set of lights a layer is left out of draws a light picture
// of its own on the GPU, up to four a window with ShadedBy's; past that,
// a layer falls back to the window's picture, lit by every light, which
// the F3 overlay and a log line tell. A Static light lights everything.
func Affects(layers ...*twod.Layer) Option { return func(l *Light) { l.SetAffects(layers...) } }

// Affects are the layers the light lights, nil for every lit one.
func (l *Light) Affects() []*twod.Layer { return slices.Clone(l.affects) }

// SetAffects changes the layers the light lights, none for every one.
func (l *Light) SetAffects(layers ...*twod.Layer) {
	l.affects = slices.Clone(layers)
	for _, layer := range layers {
		state(layer).Named = true
	}
}

// ShadowColor is the colour of the light's shadows, black by default:
// where they fall, its light is let through times c, so a dark blue
// gives blue shadows under a warm lamp. Every shadow of the light takes
// it: boxes, tiles, silhouettes, a directional light's.
func ShadowColor(c color.Color) Option { return func(l *Light) { l.shadowColor = c } }

// ShadowOpacity is how much of the light its shadows take, 0 to 1: 1, the
// default, all of it but its ShadowColor; 0.5 lets half through.
func ShadowOpacity(f float32) Option { return func(l *Light) { l.shadowOpacity = clamp01(f) } }

// Size is the light's own size, in points, and its shadows' penumbra:
// their edges are sharp where they leave what casts them and soften with
// distance behind it; 0, the default, keeps them sharp.
func Size(points float32) Option { return func(l *Light) { l.size = max(points, 0) } }

// Fade is how a shadow fades out along its length.
type Fade int

const (
	// FadeNone keeps shadows full to their end.
	FadeNone Fade = iota
	// FadeLinear fades them evenly.
	FadeLinear
	// FadeSmooth fades them slowly at both ends (smoothstep).
	FadeSmooth
	// FadeSoft fades them fast near what casts them, then over a long,
	// faint tail (ease-out, quadratic).
	FadeSoft
)

// String names the curve: off, linear, smooth or soft.
func (f Fade) String() string {
	switch f {
	case FadeLinear:
		return "linear"
	case FadeSmooth:
		return "smooth"
	case FadeSoft:
		return "soft"
	case FadeNone:
	}
	return "off"
}

// ShadowFade fades the light's shadows out along their length, dark at
// what casts them, gone at their end, by curve: FadeLinear when none is
// given, FadeNone for none, so a setting can turn it off.
func ShadowFade(curve ...Fade) Option {
	return func(l *Light) {
		l.fade = FadeLinear
		if len(curve) > 0 {
			l.fade = curve[0]
		}
	}
}

// Static draws a point or cone light once, on the GPU, into tiles of its
// camera's world, with the shadows of the camera's OccludersFrom solids
// and tile maps, and again only where it or a shape near it changes: a
// torch on a wall costs nothing a frame, however many there are. It
// counts against no budget and fills no shadow map row a frame. Bodies
// and Occluder sprites, which move, cast no shadow from it. A shape or
// directional light ignores it.
func Static() Option { return func(l *Light) { l.static = true } }

// FadeFrom keeps a fading shadow full over share of its length, 0 to 1,
// then fades it by its curve over the rest.
func FadeFrom(share float32) Option { return func(l *Light) { l.fadeFrom = clamp01(share) } }

// ShadowLength is how far a directional light's shadows reach, in points,
// 256 by default.
func ShadowLength(points float32) Option { return func(l *Light) { l.length = max(points, 0) } }

func groupBits(groups []int) uint8 {
	var bits uint8
	for _, g := range groups {
		if g >= 0 && g < draw.OccluderGroups {
			bits |= 1 << g
		}
	}
	return bits
}

func (l *Light) Radius() float32          { return l.radius }
func (l *Light) SetRadius(radius float32) { l.radius = max(radius, 0) }
func (l *Light) Color() color.Color       { return l.color }
func (l *Light) SetColor(c color.Color)   { l.color = c }
func (l *Light) Falloff() float32         { return l.falloff }
func (l *Light) SetFalloff(full float32)  { l.falloff = clamp01(full) }
func (l *Light) Intensity() float32       { return l.intensity }
func (l *Light) SetIntensity(i float32)   { l.intensity = max(i, 0) }
func (l *Light) Shadows() bool            { return !l.xray }
func (l *Light) SetShadows(on bool)       { l.xray = !on }

// BlockedBy is the occluder groups that cast the light's shadows.
func (l *Light) BlockedBy() []int { return groupList(l.groups) }

// groupList is the groups set in bits, in order.
func groupList(bits uint8) []int {
	var groups []int
	for g := range draw.OccluderGroups {
		if bits&(1<<g) != 0 {
			groups = append(groups, g)
		}
	}
	return groups
}
func (l *Light) SetBlockedBy(groups ...int) { l.groups = groupBits(groups) }

// Angle is where a cone shines or a directional light's rays go, in
// radians; Spread is a cone's width.
func (l *Light) Angle() float32            { return l.angle }
func (l *Light) SetAngle(radians float32)  { l.angle = radians }
func (l *Light) Spread() float32           { return l.spread }
func (l *Light) SetSpread(radians float32) { l.spread = radians }
func (l *Light) Elevation() float32        { return l.elevation }
func (l *Light) SetElevation(p float32)    { l.elevation = max(p, 0) }
func (l *Light) Size() float32             { return l.size }
func (l *Light) SetSize(p float32)         { l.size = max(p, 0) }
func (l *Light) ShadowFade() Fade          { return l.fade }
func (l *Light) SetShadowFade(curve Fade)  { l.fade = curve }
func (l *Light) FadeFrom() float32         { return l.fadeFrom }
func (l *Light) Static() bool              { return l.static }
func (l *Light) SetStatic(on bool)         { l.static = on }
func (l *Light) SetFadeFrom(share float32) { l.fadeFrom = clamp01(share) }

func (l *Light) ShadowColor() color.Color     { return l.shadowColor }
func (l *Light) SetShadowColor(c color.Color) { l.shadowColor = c }
func (l *Light) ShadowOpacity() float32       { return l.shadowOpacity }
func (l *Light) SetShadowOpacity(f float32)   { l.shadowOpacity = clamp01(f) }

// shadowTint is how much of l's light its shadows let through, by
// channel.
func (l *Light) shadowTint() color.Color {
	through := func(c float32) float32 { return 1 - l.shadowOpacity*(1-c) }
	return color.RGB(through(l.shadowColor.R), through(l.shadowColor.G), through(l.shadowColor.B))
}
func (l *Light) ShadowLength() float32     { return l.length }
func (l *Light) SetShadowLength(p float32) { l.length = max(p, 0) }

// Shape is a shape light's shape; SetShape changes it.
func (l *Light) Shape() Shape     { return l.shape }
func (l *Light) SetShape(s Shape) { l.shape = s }

func clamp01(v float32) float32 { return min(max(v, 0), 1) }
