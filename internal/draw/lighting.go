package draw

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/render"
	"github.com/merlindorin/ember/world"
)

// LightScale is a window's light picture's resolution, as a share of its
// pixels.
const LightScale = 0.5

// LightMargin is how far past its window, in points, a window's light and
// occluder pictures reach: a lit sprite whose anchor or pixels stand just
// outside still finds its light, and a light just outside still casts.
const LightMargin = 64

// OccluderScale is the occluder picture's texels per point.
const OccluderScale = 1

// LightRect is the part of a window of size points, in its points, that
// its light and occluder pictures cover.
func LightRect(size geom.Vec2) geom.Rect {
	return geom.Rect{X: -LightMargin, Y: -LightMargin, W: size.X + 2*LightMargin, H: size.Y + 2*LightMargin}
}

// ToLight maps a window's points to its light picture's 0..1.
func ToLight(size geom.Vec2) geom.Transform2D {
	r := LightRect(size)
	return geom.Transform2D{X: geom.V2(1/r.W, 0), Y: geom.V2(0, 1/r.H), Origin: geom.V2(-r.X/r.W, -r.Y/r.H)}
}

// Lighting is what lights one window this frame: the light every spot
// gets anyway, the lights in reach of its view and what casts their
// shadows, all in the window's points.
type Lighting struct {
	Ambient color.Color
	Lights  []Light
	// Budget is the camera's most lights, 0 for none, and Fading how
	// many of Lights are fading in or out past it.
	Budget, Fading int
	// Exposure scales the light every lit drawing reads, 0 or 1 as is:
	// the player's brightness, never the light probes read.
	Exposure float32
	// Plan is the light pictures the window draws besides its own, for
	// ShadedBy and Affects.
	Plan PicturePlan
	// Occluders are the shapes that block light, drawn in points from
	// LightRect's top-left corner, each in the colour of its groups.
	Occluders *List
	// Emissive is the glows of emissive sprites, added to the light in
	// the same points; each shows a texture Glows says how to make.
	Emissive *List
	Glows    map[*render.Texture]Glow
	// Probes are the points the game asks the light at this frame.
	Probes []Probe
	// AmbientMap is the ambient picture's tiles in view, in the same points:
	// each spot's ambient light, over Ambient where none is painted.
	AmbientMap *List
	// Debug is drawn over the window after the debug view, in its
	// points: F6's outlines.
	Debug *List
	// Static is the window's static lights, baked into tiles of the
	// world; nil for none.
	Static *StaticLights
	// Silhouettes are the lights casting the shadows of Silhouette
	// sprites, their pictures projected away from each. A directional
	// light's are drawn into its List, in this Lighting's points; another
	// light's are SilhouetteQuads, projected on the GPU into its tile of a
	// mask atlas page SilhouetteSide points square.
	Silhouettes     []Silhouette
	SilhouetteQuads []SilhouetteQuad
	SilhouetteSide  float32
}

// SilhouetteQuad is a sprite's picture as one light casts it: Set is the
// Silhouettes entry of the light, Groups the sprite's occluder groups, UV
// the part of Texture it shows (u0, v0, u1, v1, flipped by order), Rect
// its rectangle in its own space, Foot where it stands in the Lighting's
// points, Across how wide one of its points is there and Height how tall
// it stands, in points.
type SilhouetteQuad struct {
	Set     int
	Groups  uint8
	Texture *render.Texture
	UV      [4]float32
	Rect    geom.Rect
	Foot    geom.Vec2
	Across  float32
	Height  float32
}

// Silhouette is the shadows sprites of the occluder groups Groups cast
// from Lights[Light]. Area, in the Lighting's points, is where they can
// fall: its reach within the light rectangle, drawn at Tile of mask atlas
// page Page, Area's top-left at Tile's. A directional light's are drawn
// into List instead.
type Silhouette struct {
	Light  int
	Groups uint8
	Area   geom.Rect
	Page   int
	Tile   geom.Rect
	List   *List
}

// StaticLights are a window's static lights, drawn once into tiles of its
// camera's world that the GPU keeps, again only when a tile's Version
// changes, and added to the light picture where the tiles show.
type StaticLights struct {
	// Key tells bakes apart: the tiles of a camera are kept under it.
	Key any
	// TileSize is a tile's side, in the world's points; a tile covers
	// X·TileSize, Y·TileSize to the next.
	TileSize float32
	// ToWindow maps the world to the window's points.
	ToWindow geom.Transform2D
	// Tiles are the tiles in view that a static light reaches.
	Tiles []StaticTile
	// Count is how many static lights there are.
	Count int
}

// StaticTile is a tile of the world in view. Bake gives what to draw it
// with, called only when the GPU hasn't drawn this Version of it.
type StaticTile struct {
	X, Y    int
	Version uint64
	Bake    func() TileBake
}

// TileBake is a tile drawn as a window of Size points would be, its
// Lighting in that window's points: the tile is its Inner rectangle, and
// the rest the reach of its lights round it, so what blocks them is there.
type TileBake struct {
	Size     geom.Vec2
	Inner    geom.Rect
	Lighting *Lighting
}

// Probe is a point a game asks the light at, At in its window's points:
// Result gets the light there, as the GPU drew it, a few frames later.
type Probe struct {
	At     geom.Vec2
	Result *ProbeResult
}

// ProbeResult is the light a probe found, once Ready; Stale while the
// probe stands out of view, keeping what it found last.
type ProbeResult struct {
	Light color.Color
	Ready bool
	Stale bool
}

// MaxProbes is how many probes a window answers a frame.
const MaxProbes = 1024

// Glow is how to make an emissive sprite's glow texture: its picture,
// Region of Source in pixels, blurred over Pad pixels round it.
type Glow struct {
	Source *render.Texture
	Region geom.Rect
	Pad    float32
}

// LightKind is what a light is: a disc round a point, a slice of one, a
// light across the whole view, or one round a shape.
type LightKind int

const (
	PointLight LightKind = iota
	ConeLight
	DirectionalLight
	ShapeLight
)

// ShapeKind is a shape light's shape.
type ShapeKind int

const (
	RectShape ShapeKind = iota
	CircleShape
	CapsuleShape
	LineShape
	PolygonShape
)

// Light is one light, in its window's points: a soft disc of Radius round
// At, full out to Falloff of it, its colour already times its intensity.
// An XRay light casts no shadow; the others cast the shadows of the
// occluder groups in Groups.
//
// A cone shines along Dir, Spread radians wide. A directional light
// crosses the whole view along Dir, its shadows Length points long. A
// shape light is full inside its Shape and fades over Radius past it: a
// rectangle of Size centred on At turned to Dir, a circle or a capsule of
// Size.X round Points, a line along Points, or the polygon Points.
type Light struct {
	Kind    LightKind
	At      geom.Vec2
	Radius  float32
	Falloff float32
	Color   color.Color
	XRay    bool
	Groups  uint8
	Dir     geom.Vec2
	Spread  float32
	Length  float32
	Shape   ShapeKind
	Size    geom.Vec2
	Points  []geom.Vec2
	// Samples are where a shape light casts its shadows from, points
	// along its outline.
	Samples []geom.Vec2
	// Elevation is how high the light stands, in points: an occluder of
	// height h, d away, shadows out to d·Elevation/(Elevation−h); 0 for
	// shadows to its radius. ShadowSize is the light's own size, its
	// shadows' edges softening with distance behind what casts them.
	// ShadowFade fades a shadow out along its length by a curve, 0 for
	// none, then linear, smooth and soft, from FadeFrom of it.
	Elevation  float32
	ShadowSize float32
	ShadowFade int
	FadeFrom   float32
	// Affects are the layers it lights, and what they hold, nested layers
	// too; nil for every lit layer.
	Affects []any
	// ShadowTint is how much of its light its shadows let through, by
	// channel: black, the zero value, for none; its ShadowColor times its
	// ShadowOpacity over what opacity leaves.
	ShadowTint color.Color
	// Unblocked is set when no occluder of its groups lies within its
	// reach: it casts no shadow, as if XRay, and costs none.
	Unblocked bool
}

// OccluderGroups is how many groups of occluders a light can tell apart:
// the occluder picture's colour channels.
const OccluderGroups = 3

// GroupColor is the colour an occluder of groups is drawn in: a colour
// channel a group, the first in red, opaque.
func GroupColor(groups uint8) color.Color {
	on := func(g uint8) float32 {
		if groups&(1<<g) != 0 {
			return 1
		}
		return 0
	}
	return color.Color{R: on(0), G: on(1), B: on(2), A: 1}
}

// MaxOccluderHeight is the tallest an occluder's height reads, in points:
// the occluder picture keeps heights as a share of it.
const MaxOccluderHeight = 1024

// OccluderColor is the colour an occluder of groups standing height
// points tall is drawn in: GroupColor, its alpha its height as a share of
// MaxOccluderHeight, 1 for an occluder of no height given, which blocks
// all the way.
func OccluderColor(groups uint8, height float32) color.Color {
	c := GroupColor(groups)
	if height > 0 {
		c.A = min(height/MaxOccluderHeight, 254.0/255)
	}
	return c
}

// MaxShadowLights is how many lights in a window cast shadows; x-ray
// lights don't count, and those past it light as x-ray.
const MaxShadowLights = 256

// LightSource is what lights each window: the light package installs one.
type LightSource interface {
	Lighting(win *platform.Window) *Lighting
}

type lightSource struct{ LightSource }

// InstallLighting makes s the World's source of lighting.
func InstallLighting(w *world.World, s LightSource) { slot.Set(w, &lightSource{s}) }

// LightingFor is what lights win this frame, nil when nothing does.
func LightingFor(w *world.World, win *platform.Window) *Lighting {
	if s := slot.Get[lightSource](w); s != nil {
		return s.Lighting(win)
	}
	return nil
}

// ShapeBounds is the rectangle round a shape light's shape, in its
// window's points, without its reach.
func (l Light) ShapeBounds() geom.Rect {
	var pts []geom.Vec2
	grow := float32(0)
	switch l.Shape {
	case RectShape:
		hx, hy := l.Dir.Scale(l.Size.X/2), geom.V2(-l.Dir.Y, l.Dir.X).Scale(l.Size.Y/2)
		pts = []geom.Vec2{l.At.Add(hx).Add(hy), l.At.Add(hx).Sub(hy), l.At.Sub(hx).Add(hy), l.At.Sub(hx).Sub(hy)}
	case CircleShape, CapsuleShape:
		pts, grow = l.Points, l.Size.X
	case LineShape, PolygonShape:
		pts = l.Points
	}
	if len(pts) == 0 {
		return geom.Rect{X: l.At.X, Y: l.At.Y}
	}
	lo, hi := pts[0], pts[0]
	for _, p := range pts[1:] {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	return geom.Rect{X: lo.X - grow, Y: lo.Y - grow, W: hi.X - lo.X + 2*grow, H: hi.Y - lo.Y + 2*grow}
}

// LightView is what the lighting's debug view shows over every lit
// window: nothing, the light picture, the occluder picture (a colour a
// group, brighter for taller) or the shadow maps (a row a shadow-casting
// light, its angles across, dark where a shadow starts near it).
type LightView int

const (
	LightViewOff LightView = iota
	LightViewPicture
	LightViewOccluders
	LightViewShadows
)

// String names the view.
func (v LightView) String() string {
	switch v {
	case LightViewPicture:
		return "light picture"
	case LightViewOccluders:
		return "occluders"
	case LightViewShadows:
		return "shadow maps"
	case LightViewOff:
	}
	return "off"
}

// LightDebug is a window's lighting debug view: what shows over it,
// whether occluders are outlined, and where, in its points; all of it
// when Area is empty.
type LightDebug struct {
	View     LightView
	Outlines bool
	Area     geom.Rect
}

type lightDebugs struct {
	byWin map[*platform.Window]LightDebug
	panel geom.Rect
}

func debugsOf(w *world.World) *lightDebugs {
	d := slot.Get[lightDebugs](w)
	if d == nil {
		d = &lightDebugs{byWin: map[*platform.Window]LightDebug{}}
		slot.Set(w, d)
	}
	return d
}

// SetLightDebug sets win's lighting debug view in w.
func SetLightDebug(w *world.World, win *platform.Window, d LightDebug) {
	if d == (LightDebug{}) {
		delete(debugsOf(w).byWin, win)
		return
	}
	debugsOf(w).byWin[win] = d
}

// LightDebugOf is win's lighting debug view in w.
func LightDebugOf(w *world.World, win *platform.Window) LightDebug {
	if d := slot.Get[lightDebugs](w); d != nil {
		return d.byWin[win]
	}
	return LightDebug{}
}

// SetDebugPanel says where F3's panel stands in w's primary window, in
// its points, left clear of the debug view.
func SetDebugPanel(w *world.World, r geom.Rect) { debugsOf(w).panel = r }

// DebugPanel is where F3's panel stands, empty when hidden.
func DebugPanel(w *world.World) geom.Rect {
	if d := slot.Get[lightDebugs](w); d != nil {
		return d.panel
	}
	return geom.Rect{}
}
