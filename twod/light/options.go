package light

import (
	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

func state(e twod.Entity) *lit.State { return lit.Of(e) }

// Lit makes a layer's sprites, and whatever else it holds, take the light
// falling on each of their pixels.
func Lit() twod.Option { return func(e twod.Entity) { state(e).Mode = lit.PerPixel } }

// PerPixel lights an entity, and what it holds, at each of its pixels, as
// a Lit layer does. On a tile map, it lights the map's tall tiles per pixel
// too: by default they take their light just in front of their foot, as
// LitAt would, since their picture rises above the cell that blocks light;
// one that blocks light takes the brightest just past the sides of what
// it blocks with, so a light behind it lights it as one in front does.
func PerPixel() twod.Option { return func(e twod.Entity) { state(e).Mode = lit.PerPixel } }

// Unlit draws a sprite, and what it holds, as it is, whatever the light
// round it: a flame, glowing eyes.
func Unlit() twod.Option { return func(e twod.Entity) { state(e).Mode = lit.Off } }

// LitAt lights the whole of a sprite, and what it holds, as the point x,
// y of it, in its own space: a wall or a column takes its light at its
// base, evenly, never cut by a light's edge.
func LitAt(x, y float32) twod.Option {
	return func(e twod.Entity) { s := state(e); s.Mode, s.Anchor = lit.AtAnchor, geom.V2(x, y) }
}

// Ambient is the light every spot of a camera's view gets anyway, black by
// default: an option of the camera.
func Ambient(c color.Color) twod.Option {
	return func(e twod.Entity) { s := state(e); s.Ambient, s.HasAmbient = c, true }
}

// AmbientOf is the ambient light of cam's view, black unless set.
func AmbientOf(cam *twod.Camera) color.Color {
	if s := state(cam); s.HasAmbient {
		return s.Ambient
	}
	return color.Black
}

// SetAmbient changes the ambient light of cam's view, as Ambient sets it:
// tween it for day and night.
func SetAmbient(cam *twod.Camera, c color.Color) { Ambient(c)(cam) }

// Exposure scales the light a camera's view reads, 1 by default: the
// player's brightness setting, 1.3 brighter, 0.8 darker, never below
// 0.01. What is lit
// fully stays as it is; light probes read the light unscaled, so it never
// changes what the game sees. An option of the camera.
func Exposure(f float32) twod.Option {
	return func(e twod.Entity) { s := state(e); s.Exposure, s.HasExposure = max(f, 0.01), true }
}

// ExposureOf is cam's Exposure; SetExposure changes it.
func ExposureOf(cam *twod.Camera) float32 {
	if s := state(cam); s.HasExposure {
		return s.Exposure
	}
	return 1
}

func SetExposure(cam *twod.Camera, f float32) { Exposure(f)(cam) }

// AmbientMap makes the ambient light of each spot of a camera's view the
// colour of its texel in pic, a tiled picture in the world's space (its
// texel x, y covering x and y times its TexelSize), painted by the game:
// zone levels, lit rooms, halos round places, each a colour. Where pic
// paints nothing (transparent), Ambient shows. Give pic Smooth for
// gradients between texels. An option of the camera.
func AmbientMap(pic *twod.TiledPicture) twod.Option {
	return func(e twod.Entity) { state(e).AmbientMap = pic }
}

// AmbientMapOf is cam's ambient picture, nil for none; SetAmbientMap
// changes it, nil for none.
func AmbientMapOf(cam *twod.Camera) *twod.TiledPicture {
	pic, _ := state(cam).AmbientMap.(*twod.TiledPicture)
	return pic
}

func SetAmbientMap(cam *twod.Camera, pic *twod.TiledPicture) {
	if pic == nil {
		state(cam).AmbientMap = nil
		return
	}
	AmbientMap(pic)(cam)
}

// NoShadows makes a tile map's tiles cast no shadows, whatever their
// properties say: a map drawn as a picture only, whose walls something
// else stands for.
func NoShadows() twod.Option { return func(e twod.Entity) { state(e).NoShadows = true } }

// NoShadowsOf reports whether e is made with NoShadows.
func NoShadowsOf(e twod.Entity) bool { return state(e).NoShadows }

// OccludersFrom makes the solids and hard bodies of space block light in a
// camera's view, as occluder group 0: an option of the camera. Tile maps
// are not among them: light finds the World's maps itself, and a solid
// reading one, as collide/tiles' Layer does, is skipped.
func OccludersFrom(space *collide.Space) twod.Option {
	return func(e twod.Entity) { s := state(e); s.Spaces = append(s.Spaces, space) }
}

// Emissive makes a sprite's own picture glow and light round it, reach
// points past its edge, in its own colours: lava, a lit window, a rune.
// Its glow is its picture blurred once on the GPU, kept while the picture
// and reach stay the same.
func Emissive(reach float32) twod.Option {
	return func(e twod.Entity) { state(e).Emissive = max(reach, 0); world.Regroup(e) }
}

// EmissiveOf is how far e's picture glows, 0 for not at all; SetEmissive
// changes it, 0 to stop.
func EmissiveOf(e twod.Entity) float32 { return state(e).Emissive }

func SetEmissive(e twod.Entity, reach float32) { Emissive(reach)(e) }

// Occluder makes a sprite block light, over its whole rectangle, as the
// occluder groups given (0, 1 or 2; 0 when none is): a light BlockedBy
// other groups shines through it. On a Lit layer it takes the brightest
// light just past its left, right and bottom sides, never its own shadow,
// as a solid tile does; PerPixel on the sprite lights it per pixel instead.
func Occluder(groups ...int) twod.Option {
	return func(e twod.Entity) {
		s := state(e)
		s.Occluder, s.Groups = true, groupBits(groups)
		if s.Groups == 0 {
			s.Groups = 1
		}
		world.Regroup(e)
	}
}

// OccluderOf is the occluder groups e blocks light for, and whether it
// blocks light at all, by its box or its Silhouette.
func OccluderOf(e twod.Entity) ([]int, bool) {
	s := state(e)
	return groupList(s.Groups), s.Occluder
}

// SetOccluder makes e block light by its box, as the groups given, as
// Occluder does: a Silhouette sprite casts its box again.
func SetOccluder(e twod.Entity, groups ...int) {
	state(e).Silhouette = false
	Occluder(groups...)(e)
}

// RemoveOccluder stops e blocking light, by its box or its Silhouette.
func RemoveOccluder(e twod.Entity) {
	s := state(e)
	s.Occluder, s.Silhouette, s.Groups = false, false, 0
	world.Regroup(e)
}

// Height is how tall an Occluder sprite stands, in points of its own
// space, for the length of its shadows from a light with an Elevation;
// 0, the default, blocks all the way.
func Height(points float32) twod.Option {
	return func(e twod.Entity) { state(e).Height = max(points, 0) }
}

// HeightOf is e's Height; SetHeight changes it.
func HeightOf(e twod.Entity) float32 { return state(e).Height }

func SetHeight(e twod.Entity, points float32) { Height(points)(e) }

// ShadedBy lets only the shadows of the occluder groups given (0, 1 or 2)
// fall on a lit layer or entity, and what it holds: every group's by
// default, none's with no group. A crowd ShadedBy(0, 1) is shaded by walls
// and props but never by the boxes at its creatures' feet in group 2,
// their own included; the mirror of BlockedBy on a light, which picks
// what casts its shadows. Each set of groups used in a window lights it
// once more on the GPU. A Static light's shadows fall on everything.
func ShadedBy(groups ...int) twod.Option {
	return func(e twod.Entity) { s := state(e); s.Shaded, s.ShadedSet = groupBits(groups), true }
}

// ShadedByOf is the occluder groups whose shadows fall on e, and whether
// e sets them; when it doesn't, those of what holds it fall on it.
func ShadedByOf(e twod.Entity) ([]int, bool) {
	s := state(e)
	return groupList(s.Shaded), s.ShadedSet
}

// SetShadedBy changes the occluder groups whose shadows fall on e, as
// ShadedBy sets them.
func SetShadedBy(e twod.Entity, groups ...int) { ShadedBy(groups...)(e) }

// Silhouette makes a sprite cast its own picture as its shadow: from its
// foot, the bottom of its picture, stretched away from each light that
// casts on it, as long as it is tall (its Height, else its picture's),
// longer the nearer a light with an Elevation stands to its height; a
// directional light casts it ShadowLength long. Only the two or three
// strongest lights reaching it cast it, and directional lights; an
// animated sprite casts its current frame. It occludes as the groups given
// (0 when none is), for BlockedBy and ShadedBy, but blocks no light with
// its box. Static lights don't cast it.
func Silhouette(groups ...int) twod.Option {
	return func(e twod.Entity) {
		s := state(e)
		s.Occluder, s.Silhouette = true, true
		if g := groupBits(groups); g != 0 || s.Groups == 0 {
			s.Groups = max(g, 1)
		}
		world.Regroup(e)
	}
}

// SetSilhouette makes e cast its picture as its shadow, as Silhouette
// does; SetOccluder turns it back to its box, RemoveOccluder off.
func SetSilhouette(e twod.Entity, groups ...int) { Silhouette(groups...)(e) }

// SilhouetteOf reports whether e casts its picture as its shadow.
func SilhouetteOf(e twod.Entity) bool { return state(e).Silhouette }
