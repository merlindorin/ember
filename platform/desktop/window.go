package desktop

import (
	"image"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
)

func flag(on bool) int {
	if on {
		return glfw.True
	}
	return glfw.False
}

func (n *native) SetResizable(on bool)   { n.h.SetAttrib(glfw.Resizable, flag(on)) }
func (n *native) SetAlwaysOnTop(on bool) { n.h.SetAttrib(glfw.Floating, flag(on)) }

func (n *native) SetSizeLimits(lo, hi geom.Vec2) {
	limit := func(v geom.Vec2) (int, int) {
		if v == (geom.Vec2{}) {
			return glfw.DontCare, glfw.DontCare
		}
		return int(v.X), int(v.Y)
	}
	minW, minH := limit(lo)
	maxW, maxH := limit(hi)
	n.h.SetSizeLimits(minW, minH, maxW, maxH)
}

func (n *native) SetAspectRatio(width, height int) {
	if width <= 0 || height <= 0 {
		width, height = glfw.DontCare, glfw.DontCare
	}
	n.h.SetAspectRatio(width, height)
}

func (n *native) Maximize() { n.h.Maximize() }
func (n *native) Minimize() { n.h.Iconify() }
func (n *native) Restore()  { n.h.Restore() }

func (n *native) SetIcon(images []image.Image) { n.h.SetIcon(images) }

// SetFullscreen covers the monitor under the window's centre at its
// resolution, and puts the window back where it was when leaving: its
// position and size, maximized again if it was.
func (n *native) SetFullscreen(on bool) {
	switch {
	case on && n.h.GetMonitor() == nil:
		m := n.monitor()
		n.windowedMaximized = n.h.GetAttrib(glfw.Maximized) == glfw.True
		if n.windowedMaximized {
			n.h.Restore()
		}
		n.windowedX, n.windowedY = n.h.GetPos()
		n.windowedW, n.windowedH = n.h.GetSize()
		mode := m.GetVideoMode()
		n.h.SetMonitor(m, 0, 0, mode.Width, mode.Height, mode.RefreshRate)
	case !on && n.h.GetMonitor() != nil:
		n.h.SetMonitor(nil, n.windowedX, n.windowedY, n.windowedW, n.windowedH, 0)
		if n.windowedMaximized {
			n.h.Maximize()
		}
	}
	n.sync()
	n.window.SyncState(n.h.GetAttrib(glfw.Maximized) == glfw.True, false)
}

func (n *native) monitor() *glfw.Monitor {
	x, y := n.h.GetPos()
	w, h := n.h.GetSize()
	cx, cy := x+w/2, y+h/2
	for _, m := range glfw.GetMonitors() {
		mx, my, mw, mh := m.GetWorkarea()
		if cx >= mx && cx < mx+mw && cy >= my && cy < my+mh {
			return m
		}
	}
	return glfw.GetPrimaryMonitor()
}

// hints sets the creation hints for what a window asks for.
func hints(win *platform.Window) {
	glfw.WindowHint(glfw.Resizable, flag(win.Resizable()))
	glfw.WindowHint(glfw.Floating, flag(win.AlwaysOnTop()))
	glfw.WindowHint(glfw.Maximized, flag(win.Maximized()))
}

// apply sets what can only be set once the window exists.
func (n *native) apply(win *platform.Window) {
	if lo, hi := win.MinSize(), win.MaxSize(); lo != (geom.Vec2{}) || hi != (geom.Vec2{}) {
		n.SetSizeLimits(lo, hi)
	}
	if w, h := win.AspectRatio(); w > 0 && h > 0 {
		n.SetAspectRatio(w, h)
	}
	if icon := win.Icon(); len(icon) > 0 {
		n.SetIcon(icon)
	}
	if win.Fullscreen() {
		n.SetFullscreen(true)
	}
	n.h.SetMaximizeCallback(func(_ *glfw.Window, maximized bool) { win.SyncState(maximized, win.Minimized()) })
	n.h.SetIconifyCallback(func(_ *glfw.Window, minimized bool) { win.SyncState(win.Maximized(), minimized) })
}
