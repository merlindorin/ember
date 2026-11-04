package twod

import (
	"fmt"
	"hash/fnv"
	"slices"

	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/merlindorin/ember/internal/pointer"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

// Canvas is what 2D entities and ui elements draw on: a window's shapes,
// images and text, with its input, in the space of what is being drawn.
// A game's own element draws on it in its Draw:
//
//	func (o *Orb) Draw(c *twod.Canvas, name string, r geom.Rect) {
//		c.Orb(r.Center(), min(r.W, r.H)/2, o.health/o.max, red)
//		c.Record(name, automation.RoleMeter, "Health", o.health, true, r)
//	}
//
// For returns the canvas of another window, with its own draw list and
// input.
type Canvas struct {
	// list is what is being drawn on now; back is what negative layers
	// draw, behind the 3D scene.
	list   *draw.List
	back   *draw.List
	in     *input.State
	win    *platform.Window
	hot    uint64
	active uint64
	// focus is the element holding the keyboard.
	focus   uint64
	windows *windows
	scope   []string
	size    geom.Vec2
	// passes are drawn before the window; pass is the one being drawn on,
	// and scaled the camera whose Resolution pass it is, if any.
	passes []*draw.Pass
	// pictures are the layers' passes shown this frame, cropped once drawn.
	pictures []picture
	pass     *draw.Pass
	scaled   *Camera
	cursor   platform.Cursor
	focused  bool
	text     *Text
	// pending is the entity whose scope path is built on demand into
	// scopeBuf.
	pending  node.Noded
	scopeBuf []string
	// drawing is the entity being drawn, which takes the pointer only
	// where nothing covers it.
	drawing node.Noded
	// idle is the input of what is drawn under an owner of the window.
	idle *input.State
	// classes are the window's light classes this frame, from 1: the
	// layers some light Affects that each is under, the innermost first.
	classes [][]any
}

// ShowCursor asks for the pointer's look this frame, such as
// platform.CursorHand while what is drawn is hovered. The window shows the
// last one asked for, unless the game set its own.
func (u *Canvas) ShowCursor(c platform.Cursor) { u.cursor = c }

// SetFocused marks what is recorded next as holding its window's focus,
// for automation snapshots, until set back to false.
func (u *Canvas) SetFocused(on bool) { u.focused = on }

// Size is the size of what is being drawn, such as a ui element's, in its
// own space.
func (u *Canvas) Size() geom.Vec2 { return u.size }

// SetSize sets the size of what is drawn next; Preparers use it.
func (u *Canvas) SetSize(s geom.Vec2) { u.size = s }

// Input is the keyboard and mouse of the canvas's window, nil without one.
// While something else owns the window's input, as an open modal does
// (decision 0053), what is drawn gets an idle input instead: no keys, no
// buttons, the pointer far away.
func (u *Canvas) Input() *input.State {
	if u.in != nil && !u.takes() {
		if u.idle == nil {
			u.idle = &input.State{}
			u.idle.SetMouse(geom.V2(-1e9, -1e9))
			inputfeed.Of(u.idle).BeginFrame()
		}
		return u.idle
	}
	return u.in
}

// Drawing is the entity being drawn, nil between entities.
func (u *Canvas) Drawing() node.Noded { return u.drawing }

// takes reports whether what is drawn takes the window's input: nothing
// owns it, or what is drawn is its owner.
func (u *Canvas) takes() bool {
	if u.win == nil || u.drawing == nil {
		return true
	}
	return wm.Takes(u.windows.world, u.win, u.drawing)
}

// takesPointer reports whether what is drawn also takes the pointer:
// it takes the input, and no layer drawn over it covered the window
// under the pointer last frame (decision 0055).
func (u *Canvas) takesPointer() bool {
	if u.win == nil || u.drawing == nil {
		return true
	}
	return wm.TakesPointer(u.windows.world, u.win, u.drawing)
}

// Window is the window the canvas draws in, nil before one is known.
func (u *Canvas) Window() *platform.Window { return u.win }

// Focus gives the keyboard to the element named name in the current scope,
// such as a text input that was clicked.
func (u *Canvas) Focus(name string) { u.focus = id("focus", u.ID(name)) }

// Focused reports whether the element named name holds the keyboard.
func (u *Canvas) Focused(name string) bool { return u.focus != 0 && u.focus == id("focus", u.ID(name)) }

// Blur takes the keyboard back from whichever element holds it.
func (u *Canvas) Blur() { u.focus = 0 }

// Push opens an ID scope: elements drawn until Pop get IDs prefixed by it,
// so automation clients can address "hud/craft/stone-axe/craft".
func (u *Canvas) Push(id string) { u.path(); u.scope = append(u.scope, automation.Slug(id)) }

// Pop closes the last ID scope.
func (u *Canvas) Pop() { u.scope = u.scope[:len(u.scope)-1] }

// ID returns the element ID a widget with this label gets in the current scope.
func (u *Canvas) ID(label string) string {
	u.path()
	return automation.Join(append(append([]string(nil), u.scope...), automation.Slug(label))...)
}

// path builds the scope of the entity being drawn, from the names of the
// nodes above it, the first time an ID needs it.
func (u *Canvas) path() {
	e := u.pending
	if e == nil {
		return
	}
	u.pending = nil
	u.scopeBuf = u.scopeBuf[:0]
	for p := e.GetSceneNode().ParentNode(); p != nil; p = p.ParentNode() {
		u.scopeBuf = append(u.scopeBuf, automation.Slug(p.Name()))
	}
	slices.Reverse(u.scopeBuf)
	u.scope = u.scopeBuf
}

func (u *Canvas) registry() *automation.Registry {
	return slot.Get[automation.Registry](u.windows.world)
}

func (u *Canvas) record(e automation.Element) {
	if r := u.registry(); r.Recording() {
		e.Bounds = draw.Bounds(u.list.Transform(), e.Bounds)
		if u.scaled != nil {
			e.Bounds = draw.Bounds(u.scaled.screen, e.Bounds)
		}
		if u.win != nil {
			e.Window = u.win.Title()
		}
		e.Focused = e.Focused || u.focused
		r.Record(e)
	}
}

// Expose publishes a custom-drawn element (an orb, a minimap marker) so
// automation clients can read it. id is scoped like widget IDs.
func (u *Canvas) Expose(id, role, label string, value any, bounds geom.Rect) {
	if !u.registry().Recording() {
		return
	}
	u.record(automation.Element{ID: u.ID(id), Role: role, Label: label, Value: value, Enabled: true, Bounds: bounds})
}

type windows struct {
	world   *world.World
	primary *Canvas
	byWin   map[*platform.Window]*Canvas
	open    *windowIndex
	overlay debugOverlay
}

// CanvasFor is the canvas of win, the primary window's when nil, or nil
// before the 2D plugin is added. A game draws on it only through its
// entities' Draw; it reads the window's screen and input from it.
func CanvasFor(w *world.World, win *platform.Window) *Canvas {
	ws := slot.Get[windows](w)
	if ws == nil {
		return nil
	}
	return ws.primary.For(win)
}

// For returns the Canvas context drawing on win (nil means the primary window).
func (u *Canvas) For(win *platform.Window) *Canvas {
	ws := u.windows
	ws.bindPrimary()
	win = wm.Resolve(ws.world, win)
	if c, ok := ws.byWin[win]; ok {
		return c
	}
	c := &Canvas{list: &draw.List{}, win: win, windows: ws}
	if win != nil {
		c.in = win.Input()
	}
	ws.byWin[win] = c
	return c
}

// canvas is win's canvas, nil when nothing was ever drawn there.
func (ws *windows) canvas(win *platform.Window) *Canvas {
	ws.bindPrimary()
	return ws.byWin[wm.Resolve(ws.world, win)]
}

// Front, Back and Passes make the windows the World's draw.Source.
func (ws *windows) Front(win *platform.Window) *draw.List {
	if c := ws.canvas(win); c != nil {
		return c.list
	}
	return nil
}

func (ws *windows) Classes(win *platform.Window) [][]any {
	if c := ws.canvas(win); c != nil {
		return c.classes
	}
	return nil
}

func (ws *windows) Back(win *platform.Window) *draw.List {
	if c := ws.canvas(win); c != nil {
		return c.back
	}
	return nil
}

func (ws *windows) ToScreen(e any) (geom.Vec2, bool) {
	if flat, ok := e.(Entity); ok {
		return toScreen(ws.world, flat)
	}
	return geom.Vec2{}, false
}

func (ws *windows) Passes(win *platform.Window) []*draw.Pass {
	if c := ws.canvas(win); c != nil {
		return c.passes
	}
	return nil
}

// bindPrimary attaches the primary context to the primary window once the
// platform has created it.
func (ws *windows) bindPrimary() {
	if ws.primary.win != nil {
		return
	}
	if win := wm.Resolve(ws.world, nil); win != nil {
		delete(ws.byWin, nil)
		ws.primary.win, ws.primary.in = win, win.Input()
		ws.byWin[win] = ws.primary
	}
}

// reset clears every canvas for the new frame and forgets closed windows.
func (ws *windows) reset(time float32) {
	ws.bindPrimary()
	for win, c := range ws.byWin {
		if c != ws.primary && !ws.open.has(win) {
			delete(ws.byWin, win)
			continue
		}
		c.list.Reset(time)
		if c.back != nil {
			c.back.Reset(time)
		}
		c.scope = c.scope[:0]
		c.passes = c.passes[:0]
		c.classes = c.classes[:0]
		c.cursor = platform.CursorArrow
	}
}

// behind is the draw list of what shows behind the 3D scene.
func (u *Canvas) behind() *draw.List {
	if u.back == nil {
		u.back = &draw.List{}
	}
	return u.back
}

// windowIndex tracks which window components are alive.
type windowIndex struct {
	world.Group[platform.Windowed]
}

func (x *windowIndex) has(win *platform.Window) bool {
	for w := range x.All() {
		if w.GetWindow() == win {
			return true
		}
	}
	return false
}

// World is the World the Canvas draws for.
func (u *Canvas) World() *world.World { return u.windows.world }

// Screen is the drawable area in logical points, or in pixels while a
// Resolution camera's world or a render texture draws.
func (u *Canvas) Screen() geom.Rect {
	if u.pass != nil {
		return geom.Rect{W: u.pass.Screen.X, H: u.pass.Screen.Y}
	}
	s := geom.V2(1280, 720)
	if u.win != nil {
		s = u.win.Size()
	}
	return geom.Rect{W: s.X, H: s.Y}
}

// Mouse returns the cursor position in what is drawn next: through the
// inverse of the current transform, so a moved, turned or scaled element
// takes clicks where it shows. Under a layer covering the window there
// (decision 0055) the cursor is far away.
func (u *Canvas) Mouse() geom.Vec2 {
	if u.in == nil {
		return geom.Vec2{}
	}
	m := u.Input().Mouse()
	if !u.takesPointer() {
		m = geom.V2(-1e9, -1e9)
	}
	if u.scaled != nil {
		m = u.scaled.screen.Inverse().Xform(m)
	}
	return u.list.Transform().Inverse().Xform(m)
}

// framebuffer is the window's size in pixels, zero without one.
func (u *Canvas) framebuffer() geom.Vec2 {
	if u.win == nil {
		return geom.Vec2{}
	}
	return u.win.FramebufferSize()
}

// Hovered reports whether the cursor is inside r, and marks the mouse as
// captured so gameplay ignores clicks on the HUD.
func (u *Canvas) Hovered(r geom.Rect) bool {
	if u.in == nil || !u.takesPointer() || !r.Contains(u.Mouse()) || !pointer.Allowed(u.windows.world, u.drawing) {
		return false
	}
	inputfeed.Of(u.in).CapturePointer()
	return true
}

func id(parts ...any) uint64 {
	h := fnv.New64a()
	fmt.Fprint(h, parts...)
	return h.Sum64()
}

// canvasPlugin installs the DrawList, the Canvas context and the F3 debug overlay.
type canvasPlugin struct{}

func (canvasPlugin) Build(a *world.World) {
	ws := &windows{world: a, byWin: map[*platform.Window]*Canvas{}, open: &windowIndex{}}
	ws.primary = &Canvas{list: &draw.List{}, in: wm.Input(a), windows: ws}
	ws.byWin[nil] = ws.primary
	slot.Set(a, ws)
	draw.Install(a, ws)
	a.AddView(ws.open)
	overlay := &ws.overlay
	a.AddView(&overlay.gizmos)
	a.On(world.PreUpdate, "ui.begin", func(world.Frame) {
		ws.reset(a.Time().Elapsed)
	})
	a.On(world.PostUpdate, "ui.debug-overlay", func(world.Frame) { overlay.update(a) })
}
