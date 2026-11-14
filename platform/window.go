// Package platform defines windows, shared by gameplay code and platform
// backends. It has no native dependencies, so code that reads windows
// (cameras, UI layout) also builds for headless servers.
//
// A window is an entity: adding it opens it, and the first one added is the
// primary window.
//
//	win := platform.NewWindow(platform.Title("Map"), platform.Size(420, 420))
//	w.AddEntity(win)
//
// Removing the entity closes the window; closing it natively removes the
// entity and emits WindowClosed.
package platform

import (
	"image"

	"github.com/lantervale/ember/internal/cursorhint"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/world"
)

// Option sets up a window: its title, size, position and look.
type Option func(*options)

type options struct {
	title                      string
	width, height, x, y        int
	vsync, hidden, undecorated bool
	background                 color.Color
	fixed, onTop, fullscreen   bool
	maximized                  bool
	minSize, maxSize           geom.Vec2
	aspectW, aspectH           int
	icon                       []image.Image
}

func Title(title string) Option { return func(o *options) { o.title = title } }

// Size sets the window size in points, 1280×720 by default.
func Size(width, height int) Option {
	return func(o *options) { o.width, o.height = width, height }
}

// Position places the window's top-left corner on screen; by default
// the system places it.
func Position(x, y int) Option { return func(o *options) { o.x, o.y = x, y } }

// VSync presents the window at the display's refresh, on by default: a
// steady frame without tearing or waits of its own. VSync(false) runs
// frames as fast as they come, for measuring, and shows at most one a
// display refresh: the others are drawn but not shown, since the window
// compositor (macOS's at least) holds back any more and stalls the frame
// that asks (docs/design/performance.md).
// F3 and EMBER_STATS show the work a frame takes besides the frame rate,
// so the headroom under vsync shows.
func VSync(on bool) Option { return func(o *options) { o.vsync = on } }

// Hidden creates the window invisible, for offscreen rendering and
// tests.
func Hidden() Option { return func(o *options) { o.hidden = true } }

// Undecorated removes the system frame and title bar, for windows that
// draw their own.
func Undecorated() Option { return func(o *options) { o.undecorated = true } }

// Background clears windows that show no camera, such as tool windows.
func Background(c color.Color) Option { return func(o *options) { o.background = c } }

// FixedSize stops the user from resizing the window.
func FixedSize() Option { return func(o *options) { o.fixed = true } }

// MinSize keeps the window at least width×height points when the user
// resizes it.
func MinSize(width, height int) Option {
	return func(o *options) { o.minSize = geom.V2(float32(width), float32(height)) }
}

// MaxSize keeps the window at most width×height points when the user
// resizes it.
func MaxSize(width, height int) Option {
	return func(o *options) { o.maxSize = geom.V2(float32(width), float32(height)) }
}

// AspectRatio keeps the window's width to height ratio when the user
// resizes it, such as 16:9.
func AspectRatio(width, height int) Option {
	return func(o *options) { o.aspectW, o.aspectH = width, height }
}

// Maximized opens the window maximized.
func Maximized() Option { return func(o *options) { o.maximized = true } }

// Fullscreen opens the window fullscreen, on the monitor it would show
// on, at that monitor's resolution.
func Fullscreen() Option { return func(o *options) { o.fullscreen = true } }

// AlwaysOnTop keeps the window above the others, such as a tool palette.
func AlwaysOnTop() Option { return func(o *options) { o.onTop = true } }

// Icon sets the window's icon, the system picking the image closest to
// the size it needs. macOS ignores it and shows the app's icon.
func Icon(images ...image.Image) Option { return func(o *options) { o.icon = images } }

// Backend is implemented by native window providers.
type Backend interface {
	SetTitle(title string)
	SetCursor(c Cursor)
	Clipboard() string
	SetClipboard(text string)
	SetPosition(x, y int)
	SetSize(w, h int)
	SetResizable(on bool)
	SetAlwaysOnTop(on bool)
	// SetSizeLimits bounds user resizing to [lo, hi]; a zero size has no
	// limit.
	SetSizeLimits(lo, hi geom.Vec2)
	// SetAspectRatio keeps width:height while resizing; 0:0 frees it.
	SetAspectRatio(width, height int)
	Maximize()
	Minimize()
	Restore()
	SetFullscreen(on bool)
	SetIcon(images []image.Image)
	// ScreenCursor queries the pointer in screen points right now,
	// independent of queued events and in-flight window moves.
	ScreenCursor() (x, y float64)
	WindowPosition() (x, y int)
}

// Window is a native window, and an entity: children it holds, such as
// cameras and UI elements, show in it.
type Window struct {
	*world.BasicEntity
	opts      options
	cursor    Cursor
	shown     Cursor
	clipboard string

	backend     Backend
	input       *input.State
	size        geom.Vec2
	framebuffer geom.Vec2
	position    geom.Vec2
	focused     bool
	minimized   bool
	capture     string
}

// Windowed is any entity that is a window.
type Windowed interface {
	world.BasicFace
	GetWindow() *Window
}

// WindowClosed is emitted when the user closes a secondary window.
type WindowClosed struct {
	Window *Window
}

func NewWindow(opts ...Option) *Window {
	o := options{title: "ember", width: 1280, height: 720, background: color.RGB(0.015, 0.015, 0.02), vsync: true}
	for _, opt := range opts {
		opt(&o)
	}
	if o.width <= 0 || o.height <= 0 {
		o.width, o.height = 1280, 720
	}
	size := geom.V2(float32(o.width), float32(o.height))
	return &Window{
		BasicEntity: world.NewBasic(),
		opts:        o,
		size:        size,
		framebuffer: size,
		position:    geom.V2(float32(o.x), float32(o.y)),
		input:       &input.State{},
	}
}

func (w *Window) Background() color.Color { return w.opts.background }

// SetBackground changes the colour the window clears with, such as a sky
// that follows the time of day.
func (w *Window) SetBackground(c color.Color) { w.opts.background = c }
func (w *Window) VSync() bool                 { return w.opts.vsync }
func (w *Window) Hidden() bool                { return w.opts.hidden }
func (w *Window) Undecorated() bool           { return w.opts.undecorated }
func (w *Window) Resizable() bool             { return !w.opts.fixed }
func (w *Window) AlwaysOnTop() bool           { return w.opts.onTop }
func (w *Window) Fullscreen() bool            { return w.opts.fullscreen }
func (w *Window) Maximized() bool             { return w.opts.maximized }
func (w *Window) Minimized() bool             { return w.minimized }
func (w *Window) MinSize() geom.Vec2          { return w.opts.minSize }
func (w *Window) MaxSize() geom.Vec2          { return w.opts.maxSize }
func (w *Window) Icon() []image.Image         { return w.opts.icon }

// AspectRatio is the width:height kept while resizing, 0:0 when free.
func (w *Window) AspectRatio() (int, int) { return w.opts.aspectW, w.opts.aspectH }

func (w *Window) SetResizable(on bool) {
	w.opts.fixed = !on
	if w.backend != nil {
		w.backend.SetResizable(on)
	}
}

func (w *Window) SetAlwaysOnTop(on bool) {
	w.opts.onTop = on
	if w.backend != nil {
		w.backend.SetAlwaysOnTop(on)
	}
}

// SetSizeLimits bounds resizing to [lo, hi] points; a zero size has no
// limit.
func (w *Window) SetSizeLimits(lo, hi geom.Vec2) {
	w.opts.minSize, w.opts.maxSize = lo, hi
	if w.backend != nil {
		w.backend.SetSizeLimits(lo, hi)
	}
}

// SetAspectRatio keeps width:height while resizing; 0, 0 frees it.
func (w *Window) SetAspectRatio(width, height int) {
	w.opts.aspectW, w.opts.aspectH = width, height
	if w.backend != nil {
		w.backend.SetAspectRatio(width, height)
	}
}

func (w *Window) Maximize() {
	w.opts.maximized, w.minimized = true, false
	if w.backend != nil {
		w.backend.Maximize()
	}
}

// Minimize iconifies the window to the dock or task bar.
func (w *Window) Minimize() {
	w.minimized = true
	if w.backend != nil {
		w.backend.Minimize()
	}
}

// Restore brings the window back from maximized or minimized.
func (w *Window) Restore() {
	w.opts.maximized, w.minimized = false, false
	if w.backend != nil {
		w.backend.Restore()
	}
}

// SetFullscreen moves the window fullscreen on the monitor it shows on,
// or back to where it was.
func (w *Window) SetFullscreen(on bool) {
	w.opts.fullscreen = on
	if w.backend != nil {
		w.backend.SetFullscreen(on)
	}
}

func (w *Window) SetIcon(images ...image.Image) {
	w.opts.icon = images
	if w.backend != nil {
		w.backend.SetIcon(images)
	}
}

// SyncState records the window being maximized or minimized by the user.
// Called by platform implementations.
func (w *Window) SyncState(maximized, minimized bool) {
	w.opts.maximized, w.minimized = maximized, minimized
}

// SyncFullscreen records the window going full screen or back by the
// player or the page, as a browser's Escape does. Called by platform
// implementations.
func (w *Window) SyncFullscreen(on bool) { w.opts.fullscreen = on }

func (w *Window) GetWindow() *Window { return w }

// Input is the keyboard and mouse state of this window. The primary
// window's also holds the gamepads.
func (w *Window) Input() *input.State {
	if w.input == nil {
		w.input = &input.State{}
	}
	return w.input
}

// Position is the window's top-left corner in screen points.
func (w *Window) Position() geom.Vec2 { return w.position }

// SetPosition moves the window on screen.
func (w *Window) SetPosition(p geom.Vec2) {
	w.position = p
	if w.backend != nil {
		w.backend.SetPosition(int(p.X), int(p.Y))
	}
}

// SetSize resizes the window, in points.
func (w *Window) SetSize(s geom.Vec2) {
	w.size = s
	if w.backend != nil {
		w.backend.SetSize(int(s.X), int(s.Y))
	}
}

// ScreenCursor returns the pointer position in screen points, queried
// synchronously from the OS. Use it to drag windows: unlike the mouse
// position, it does not shift when the window itself moves.
func (w *Window) ScreenCursor() geom.Vec2 {
	if w.backend == nil {
		return w.position.Add(w.Input().Mouse())
	}
	x, y := w.backend.ScreenCursor()
	return geom.V2(float32(x), float32(y))
}

// CurrentPosition queries the window position from the OS, bypassing
// position events that may still be in flight.
func (w *Window) CurrentPosition() geom.Vec2 {
	if w.backend == nil {
		return w.position
	}
	x, y := w.backend.WindowPosition()
	w.position = geom.V2(float32(x), float32(y))
	return w.position
}

// SyncPosition records where the backend placed the window.
func (w *Window) SyncPosition(p geom.Vec2) { w.position = p }

// Size is the window size in logical points (what the mouse reports).
// Add makes children of the window, and returns it: entities that show in
// it, read its input and leave with it.
func (w *Window) Add(children ...world.BasicFace) *Window {
	for _, c := range children {
		w.AppendChild(c)
	}
	return w
}

func (w *Window) Size() geom.Vec2 { return w.size }

// FramebufferSize is the drawable size in pixels (2x Size on Retina).
func (w *Window) FramebufferSize() geom.Vec2 { return w.framebuffer }

// ContentScale is pixels per point.
func (w *Window) ContentScale() float32 {
	if w.size.X == 0 {
		return 1
	}
	return w.framebuffer.X / w.size.X
}

func (w *Window) Aspect() float32 {
	if w.framebuffer.Y == 0 {
		return 1
	}
	return w.framebuffer.X / w.framebuffer.Y
}

func (w *Window) Focused() bool { return w.focused }

func (w *Window) Title() string { return w.opts.title }

func (w *Window) SetTitle(title string) {
	w.opts.title = title
	if w.backend != nil {
		w.backend.SetTitle(title)
	}
}

// Cursor is the pointer's look over a window: one of the standard shapes
// below, or a picture from NewCursor.
type Cursor int

const (
	CursorArrow Cursor = iota
	CursorHidden
	CursorText
	CursorHand
	// CursorResizeEW, CursorResizeNS and CursorResizeNWSE show over edges
	// that resize sideways, up and down, and both from a corner. GLFW 3.3
	// has no diagonal shape: the desktop shows a crosshair for NWSE.
	CursorResizeEW
	CursorResizeNS
	CursorResizeNWSE
)

// Cursor is the pointer's look over the window now.
func (w *Window) Cursor() Cursor { return w.shown }

// SetCursor sets the pointer's look over the window: a standard shape,
// CursorHidden to hide it, or a picture of the game's from NewCursor.
// Until a game sets one, the UI shows CursorHand over what can be clicked
// and CursorText over text inputs; SetCursor(CursorArrow) gives the
// choice back to the UI.
func (w *Window) SetCursor(c Cursor) {
	w.cursor = c
	w.show(c)
}

//nolint:gochecknoinits // fills cursorhint's hook, keeping twod free of platform's windows
func init() {
	cursorhint.Show = func(win any, c int) {
		if w, ok := win.(*Window); ok && w != nil && w.cursor == CursorArrow {
			w.show(Cursor(c))
		}
	}
}

// show puts c on screen when it changes.
func (w *Window) show(c Cursor) {
	if c == w.shown {
		return
	}
	w.shown = c
	if w.backend != nil {
		w.backend.SetCursor(c)
	}
}

// Clipboard is the system clipboard's text.
func (w *Window) Clipboard() string {
	if w.backend == nil {
		return w.clipboard
	}
	return w.backend.Clipboard()
}

// SetClipboard puts text on the system clipboard; a window not open keeps
// it itself, for tests.
func (w *Window) SetClipboard(text string) {
	if w.backend == nil {
		w.clipboard = text
		return
	}
	w.backend.SetClipboard(text)
}

// Capture asks the backend to save the next presented frame as a PNG.
func (w *Window) Capture(path string) { w.capture = path }

// Open reports whether a native window currently backs this component.
func (w *Window) Open() bool { return w.backend != nil }

// Attach connects a native backend. Called by platform implementations.
func (w *Window) Attach(b Backend) {
	w.backend = b
	if b != nil && w.shown != CursorArrow {
		b.SetCursor(w.shown)
	}
}

// Sync updates sizes and focus. Called by platform implementations.
func (w *Window) Sync(size, framebuffer geom.Vec2, focused bool) {
	w.size, w.framebuffer, w.focused = size, framebuffer, focused
}

// TakeCapture returns and clears a pending capture request.
func (w *Window) TakeCapture() string {
	p := w.capture
	w.capture = ""
	return p
}
