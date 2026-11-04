// Package lit is what twod keeps for the light package on each 2D entity,
// and the hooks twod gives it: lighting lives in twod/light, which twod
// can't import.
package lit

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

// Mode is how an entity and what is under it read the light: as their
// parent does, per pixel, at one anchor, or not at all; Base is a tall
// tile's: at its anchor, just in front of its foot, or round it when it
// has a Spread, when what it is under is lit, else not at all.
type Mode int

const (
	Inherit Mode = iota
	PerPixel
	AtAnchor
	Off
	Base
)

// State is an entity's lighting: its Mode and Anchor, in its own space; on
// a camera, the Ambient light and the spaces whose solids block light;
// on a sprite, whether it blocks light itself.
type State struct {
	Mode   Mode
	Anchor geom.Vec2
	// Spread is, on a tall tile blocking light, the half-size round Anchor
	// of what it blocks with, grown a little: it takes the light of the
	// brightest of the four points past its sides, never its own shadow.
	Spread     geom.Vec2
	Ambient    color.Color
	HasAmbient bool
	// Spaces are a camera's *collide.Space values, kept untyped: collide's
	// tests import twod, which imports this package.
	Spaces   []any
	Occluder bool
	// Groups are the occluder groups a sprite blocks light for; Height
	// how tall it stands, in its own space, 0 for blocking all the way.
	Groups uint8
	Height float32
	// Silhouette makes an occluding sprite cast its picture as its
	// shadow, projected from its foot, instead of blocking light with its
	// box.
	Silhouette bool
	// Shaded are, when ShadedSet, the occluder groups whose shadows fall
	// on an entity and what is under it; else those of what it is under,
	// every group at the top.
	Shaded    uint8
	ShadedSet bool
	// Emissive is how far, in points, a sprite's own picture glows round
	// it, 0 for not at all.
	Emissive float32
	// AmbientMap is a camera's *twod.TiledPicture of ambient light, nil
	// for none.
	AmbientMap any
	// Exposure scales the light a camera's view reads, when HasExposure.
	Exposure    float32
	HasExposure bool
	// Named marks a layer some light Affects: what it holds draws in a
	// light class of its own.
	Named bool
	// Budget is how many lights a camera's view draws at most, 0 for
	// all; BudgetFade, when HasBudgetFade, how long one takes to fade
	// out or in past it, in seconds.
	Budget        int
	BudgetFade    float32
	HasBudgetFade bool
}

// Of is e's state, nil when e isn't a 2D entity; twod sets it.
var Of func(e any) *State //nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic

// Screen is where e drew in the last frame: its transform to its
// window's points and the window; twod sets it.
//
//nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic
var Screen func(w *world.World, e any) (geom.Transform2D, *platform.Window, bool)

// View is win's camera this frame: the camera entity, and its transform
// from the world to win's points; twod sets it.
//
//nolint:gochecknoglobals // a hook twod fills at init
var View func(w *world.World, win *platform.Window) (camera any, toScreen geom.Transform2D, ok bool)

// Tiles draws the tiles of picture, a *twod.TiledPicture, within view,
// in its space, onto list through t; twod sets it.
//
//nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic
var Tiles func(picture any, list any, view geom.Rect, t geom.Transform2D)

// Size is e's drawn size in its own space, centred on its pivot, for a
// sprite occluding light; twod sets it.
//
//nolint:gochecknoglobals // a hook another package fills at init, keeping the imports acyclic
var Size func(e any) (geom.Rect, bool)
