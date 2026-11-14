package twod

import (
	"math"
	"slices"

	"github.com/lantervale/ember/internal/draw"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/node"
	"github.com/lantervale/ember/platform"
)

//nolint:gochecknoinits // registers the package's node types on import
func init() {
	node.Register(node.Type{New: func() node.Noded { return NewCamera() }})
}

// Camera shows a window's 2D world centred on its position, or on the
// entity it follows. A window shows its highest-priority camera, and the
// window's top-left corner is the origin when it has none.
type Camera struct {
	Item
	zoom      float32   `node:"prop"`
	smoothing float32   `node:"prop"`
	bounds    geom.Rect `node:"prop"`
	snap      bool      `node:"prop"`
	priority  int       `node:"prop"`
	window    *platform.Window
	// pictures are the passes of the layers drawn as pictures in the
	// camera's render texture: its own, never the window's.
	pictures map[*Layer]*draw.Pass
	follow   interface{ GlobalPosition() geom.Vec2 }
	alpha    float32
	at       geom.Vec2
	placed   bool
	// resolution, when set, is the size in pixels the camera draws at
	// before it is scaled up to its window.
	resolution geom.Vec2 `node:"prop"`
	upscale    Filter    `node:"prop"`
	fit        bool      `node:"prop"`
	// viewSize is the world the camera keeps in view as its window
	// resizes, 0 on an axis it leaves free; scale is what that takes.
	viewSize geom.Vec2 `node:"prop"`
	scale    float32
	// layers, when set, are the only layers the camera draws.
	layers []*Layer
	offset geom.Vec2 `node:"prop"`
	pass   *draw.Pass
	into   *Texture
	// viewport is what the camera showed in the last frame, in points, and
	// screen maps it to its window.
	viewport geom.Vec2
	screen   geom.Transform2D
}

// NewCamera returns a camera at zoom 1.
func NewCamera(opts ...CameraOption) *Camera {
	c := Build(&Camera{Item: Base(), zoom: 1, scale: 1, upscale: Nearest, screen: geom.Identity2D()}, "camera")
	for _, o := range opts {
		o.camera(c)
	}
	return c
}

// In makes the camera show in win. Without it, a camera shows in the
// window it is under, else the primary: In is for a camera outside any
// window.
func In(win *platform.Window) CameraOption { return cameraOption(func(c *Camera) { c.window = win }) }

// Zoom magnifies the view: 2 shows everything twice as big.
func Zoom(z float32) CameraOption { return cameraOption(func(c *Camera) { c.SetZoom(z) }) }

// Follow centres the camera on e, such as the player's sprite. It follows
// where e is drawn: an entity moved in FixedUpdate is drawn interpolated
// between its last two steps, so the camera can sit up to a step behind
// its Position, as a test reading both will see.
func Follow(e interface{ GlobalPosition() geom.Vec2 }) CameraOption {
	return cameraOption(func(c *Camera) { c.follow = e })
}

// Smoothing eases the camera towards where it should be at speed per
// second, instead of snapping.
func Smoothing(speed float32) CameraOption {
	return cameraOption(func(c *Camera) { c.smoothing = speed })
}

// Bounds keeps the view inside r, such as the level's edges.
func Bounds(r geom.Rect) CameraOption { return cameraOption(func(c *Camera) { c.bounds = r }) }

// SnapToPixels rounds what the camera shows to whole pixels, for pixel art.
func SnapToPixels() CameraOption { return cameraOption(func(c *Camera) { c.snap = true }) }

// Priority picks between cameras showing in the same window, highest first.
func Priority(n int) CameraOption { return cameraOption(func(c *Camera) { c.priority = n }) }

// Resolution draws the camera's world at w×h pixels, scaled up by whole
// factors to fill its window and letterboxed: pixel art stays pixel
// perfect. Under it, one world unit is one of those pixels rather than a
// point of the window. Fixed layers and the UI still draw at the window's
// resolution, in points. A size of 0 draws the world at the window's
// resolution again: SetResolution(0, 0) turns it off.
func Resolution(w, h int) CameraOption {
	return cameraOption(func(c *Camera) {
		if w > 0 && h > 0 {
			c.resolution = geom.V2(float32(w), float32(h))
		} else {
			c.resolution = geom.Vec2{}
		}
	})
}

// Into makes the camera draw its window's 2D world, without fixed layers
// and the UI, into t, a NewRenderTexture, rather than the window: sprites
// showing t show its view. It is not one of the window's cameras.
func Into(t *Texture) CameraOption { return cameraOption(func(c *Camera) { c.into = t }) }

// Upscale scales a camera's Resolution up with f: Nearest (the default)
// keeps pixels sharp, Linear smooths them.
func Upscale(f Filter) CameraOption { return cameraOption(func(c *Camera) { c.upscale = f }) }

// FitWindow scales a camera's Resolution by any factor that fills its
// window, keeping the aspect ratio, instead of whole factors only.
func FitWindow() CameraOption { return cameraOption(func(c *Camera) { c.fit = true }) }

// ViewSize keeps at least w×h of the world in view whatever the size of
// the window, scaling the view to fit; 0 leaves an axis free, so
// ViewSize(0, 360) shows 360 units of height and more or less width with
// the window's shape. Without it, a bigger window shows more of the world
// at the same scale. A Resolution camera has its own size and ignores it.
func ViewSize(w, h float32) CameraOption {
	return cameraOption(func(c *Camera) { c.viewSize = geom.V2(max(w, 0), max(h, 0)) })
}

// Layers makes the camera draw only these layers and what is in them,
// such as a minimap's camera drawing markers the main view leaves out.
// Without it, a camera draws every layer. A window camera's fixed layers
// and the UI draw whatever it filters.
func Layers(layers ...*Layer) CameraOption {
	return cameraOption(func(c *Camera) { c.layers = layers })
}

func (c *Camera) Layers() []*Layer           { return c.layers }
func (c *Camera) SetLayers(layers ...*Layer) { c.layers = layers }

// shows reports whether the camera draws en: in one of its Layers, or in
// any layer without them.
func (c *Camera) shows(en entry) bool {
	if len(c.layers) == 0 {
		return true
	}
	for n := en.e.GetSceneNode(); n != nil; n = n.ParentNode() {
		if l, ok := n.Entity().(*Layer); ok && slices.Contains(c.layers, l) {
			return true
		}
	}
	return false
}

// Offset nudges the view by x, y in the world, after Follow, Smoothing
// and Bounds, and is never followed back: the game's own shake or recoil,
// set every frame.
func Offset(x, y float32) CameraOption {
	return cameraOption(func(c *Camera) { c.offset = geom.V2(x, y) })
}

func (c *Camera) Offset() (float32, float32) { return c.offset.X, c.offset.Y }
func (c *Camera) SetOffset(x, y float32)     { c.offset = geom.V2(x, y) }

func (c *Camera) Zoom() float32 { return c.zoom }

func (c *Camera) SetZoom(z float32) {
	if z > 0 {
		c.zoom = z
	}
}

func (c *Camera) SetFollow(e interface{ GlobalPosition() geom.Vec2 }) { c.follow = e }

// WorldToScreen maps p, a point of the world the camera shows, to its
// window, in points, from the last frame.
func (c *Camera) WorldToScreen(p geom.Vec2) geom.Vec2 {
	return c.screen.Mul(c.transform(c.viewport, 1)).Xform(p)
}

// ScreenToWorld maps p, a point of its window such as the mouse, to the
// world the camera shows, from the last frame.
func (c *Camera) ScreenToWorld(p geom.Vec2) geom.Vec2 {
	return c.screen.Mul(c.transform(c.viewport, 1)).Inverse().Xform(p)
}

// Center is the point at the middle of the view, from the last frame,
// Offset included.
func (c *Camera) Center() geom.Vec2 { return c.at.Add(c.offset) }

func (c *Camera) target() geom.Vec2 {
	if e, ok := c.follow.(Entity); ok {
		return drawnGlobal(e, c.alpha).Origin
	}
	if c.follow != nil {
		return c.follow.GlobalPosition()
	}
	return c.GlobalPosition()
}

// frame sets the camera up for this frame in a window of screen points and
// fb pixels: its viewport, and its pass with a Resolution.
func (c *Camera) frame(dt float32, screen, fb geom.Vec2) {
	c.viewport, c.screen, c.scale = screen, geom.Identity2D(), 1
	switch {
	case c.into != nil:
		if c.pass == nil || c.pass.Target != &c.into.Texture {
			c.pass = &draw.Pass{Target: &c.into.Texture, List: &draw.List{}}
		}
		size := c.into.Size()
		c.pass.Width, c.pass.Height, c.pass.Screen = int(size.X), int(size.Y), size
		c.viewport = size
	case c.resolution != (geom.Vec2{}):
		if c.pass == nil {
			c.pass = draw.NewPass()
		}
		c.pass.Width, c.pass.Height, c.pass.Screen = int(c.resolution.X), int(c.resolution.Y), c.resolution
		c.pass.Target.Filter = c.upscale
		c.viewport, c.screen = c.resolution, letterbox(c.resolution, screen, fb, c.fit)
	default:
		c.scale = fitScale(c.viewSize, screen)
	}
	if c.pass != nil {
		c.pass.Name = c.GetSceneNode().Name()
	}
	c.update(dt, c.viewport)
}

// update moves the view's centre for this frame, on a viewport in points.
func (c *Camera) update(dt float32, viewport geom.Vec2) {
	want := c.target()
	if !c.placed || c.smoothing <= 0 {
		c.at, c.placed = want, true
	} else {
		c.at = c.at.Lerp(want, min(c.smoothing*dt, 1))
	}
	if c.bounds.W > 0 && c.bounds.H > 0 {
		half := viewport.Scale(0.5 / (c.zoom * c.scale))
		c.at = geom.V2(
			clampCentre(c.at.X, c.bounds.X, c.bounds.W, half.X),
			clampCentre(c.at.Y, c.bounds.Y, c.bounds.H, half.Y),
		)
	}
}

func clampCentre(v, lo, size, half float32) float32 {
	if size <= 2*half {
		return lo + size/2
	}
	return min(max(v, lo+half), lo+size-half)
}

// transform maps a layer moving at parallax times the camera's speed to
// the viewport.
func (c *Camera) transform(viewport geom.Vec2, parallax float32) geom.Transform2D {
	k := c.zoom * c.scale
	origin := viewport.Scale(0.5).Sub(c.at.Add(c.offset).Scale(parallax * k))
	if c.snap {
		origin = geom.V2(float32(math.Round(float64(origin.X))), float32(math.Round(float64(origin.Y))))
	}
	return geom.Transform2D{X: geom.V2(k, 0), Y: geom.V2(0, k), Origin: origin}
}

// fitScale is the scale that keeps view, 0 on a free axis, inside screen.
func fitScale(view, screen geom.Vec2) float32 {
	k := float32(0)
	for _, a := range [][2]float32{{view.X, screen.X}, {view.Y, screen.Y}} {
		if a[0] > 0 && a[1] > 0 {
			if s := a[1] / a[0]; k == 0 || s < k {
				k = s
			}
		}
	}
	if k == 0 {
		return 1
	}
	return k
}
