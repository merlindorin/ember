package light

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

// DebugView is what a window's lighting debug view shows over it: F3's
// F5 cycles the primary window's.
type DebugView int

const (
	DebugOff DebugView = iota
	// DebugLight is the light picture: the light each spot gets.
	DebugLight
	// DebugOccluders is what blocks light, a colour a group (0 red,
	// 1 green, 2 blue), brighter for taller.
	DebugOccluders
	// DebugShadows is the shadow maps: a band a shadow-casting light,
	// its angles across, dark where a shadow starts near it.
	DebugShadows
)

// String names the view.
func (v DebugView) String() string { return draw.LightView(v).String() }

// SetDebug shows view v over win, a window in a World.
func SetDebug(win *platform.Window, v DebugView) {
	update(win, func(d *draw.LightDebug) { d.View = draw.LightView(v) })
}

// DebugOf is the view shown over win.
func DebugOf(win *platform.Window) DebugView { return DebugView(debugOf(win).View) }

// SetOutlines outlines win's occluders in their groups' colours, or
// stops: F3's F6 for the primary window.
func SetOutlines(win *platform.Window, on bool) {
	update(win, func(d *draw.LightDebug) { d.Outlines = on })
}

// OutlinesOf reports whether win's occluders are outlined.
func OutlinesOf(win *platform.Window) bool { return debugOf(win).Outlines }

// SetDebugArea limits win's debug view and outlines to r, in its
// points; an empty r shows them over all of it.
func SetDebugArea(win *platform.Window, r geom.Rect) {
	update(win, func(d *draw.LightDebug) { d.Area = r })
}

// DebugAreaOf is where win's debug view and outlines show, empty for all
// of it.
func DebugAreaOf(win *platform.Window) geom.Rect { return debugOf(win).Area }

func update(win *platform.Window, change func(*draw.LightDebug)) {
	w := world.Of(win)
	if w == nil {
		return
	}
	d := draw.LightDebugOf(w, win)
	change(&d)
	draw.SetLightDebug(w, win, d)
}

func debugOf(win *platform.Window) draw.LightDebug {
	if w := world.Of(win); w != nil {
		return draw.LightDebugOf(w, win)
	}
	return draw.LightDebug{}
}
