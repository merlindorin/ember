package twod

import (
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	lt "github.com/merlindorin/ember/internal/lit"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
)

//nolint:gochecknoinits // fills light's hooks, keeping the imports acyclic
func init() {
	lt.Of = func(e any) *lt.State {
		if en, ok := e.(Entity); ok {
			return &en.GetDrawLook().light
		}
		return nil
	}
	lt.Screen = func(w *world.World, e any) (geom.Transform2D, *platform.Window, bool) {
		en, ok := e.(Entity)
		if !ok {
			return geom.Transform2D{}, nil, false
		}
		return screenTransform(w, en)
	}
	lt.View = func(w *world.World, win *platform.Window) (any, geom.Transform2D, bool) {
		v := slot.Get[views](w)
		if v == nil {
			return nil, geom.Identity2D(), false
		}
		cam := v.cameras[win]
		if cam == nil {
			return nil, geom.Identity2D(), true
		}
		return cam, cam.screen.Mul(cam.transform(cam.viewport, 1)), true
	}
	lt.Tiles = func(picture any, list any, view geom.Rect, t geom.Transform2D) {
		p, isPicture := picture.(*TiledPicture)
		l, isList := list.(*draw.List)
		if isPicture && isList {
			p.drawTiles(l, view, t)
		}
	}
	lt.Size = func(e any) (geom.Rect, bool) {
		if s, ok := e.(interface{ Rect() geom.Rect }); ok {
			return s.Rect(), true
		}
		return geom.Rect{}, false
	}
}

// lighting resolves how en is lit from its own state and what it is under,
// layer: its mode and, at an anchor, where the anchor stands in its layer.
func (layer *entry) lighting(e Entity, global geom.Transform2D) {
	st := e.GetDrawLook().light
	if st.Named {
		layer.named = &namedLayer{layer: e, up: layer.named}
	}
	if st.ShadedSet {
		layer.left = draw.AllGroups &^ st.Shaded
	}
	switch st.Mode {
	case lt.PerPixel, lt.Off:
		layer.light, layer.around = st.Mode, geom.Rect{}
	case lt.AtAnchor:
		layer.light, layer.anchor2, layer.around = lt.AtAnchor, global.Xform(st.Anchor), geom.Rect{}
	case lt.Base:
		if layer.light == lt.PerPixel || layer.light == lt.AtAnchor {
			layer.light, layer.anchor2, layer.around = lt.AtAnchor, global.Xform(st.Anchor), geom.Rect{}
			if st.Spread != (geom.Vec2{}) {
				layer.around = boxOf(global.Xform(st.Anchor.Sub(st.Spread)), global.Xform(st.Anchor.Add(st.Spread)))
			}
		}
	case lt.Inherit:
	}
}

// boxOf is the rectangle with corners a and b.
func boxOf(a, b geom.Vec2) geom.Rect {
	lo, hi := geom.V2(min(a.X, b.X), min(a.Y, b.Y)), geom.V2(max(a.X, b.X), max(a.Y, b.Y))
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// baseFront is how far in front of its foot, in points, a tall tile takes
// its light, or past the sides of what it blocks light with: out of its
// own shadow, which it would otherwise stand in.
const baseFront = 3

// litAtBase lights s, a tall tile of a map lit as mapLight, at its base
// when its layer is lit, unless the map asked for PerPixel or Unlit, which
// it keeps wherever it is sorted; k is the
// tile's scale, from its pixels to points. When the tile blocks light with
// blocks, in s's parent's space, it takes the brightest light just past
// their box's sides, so a light behind it lights it as one in front does.
func litAtBase(s *Sprite, mapLight lt.Mode, k float32, blocks []geom.Rect) {
	st := &s.GetDrawLook().light
	if mapLight == lt.PerPixel || mapLight == lt.Off {
		st.Mode = mapLight
		return
	}
	k = max(k, 1e-6)
	st.Mode, st.Anchor, st.Spread = lt.Base, geom.V2(0, baseFront/k), geom.Vec2{}
	if len(blocks) == 0 {
		return
	}
	lo, hi := geom.V2(blocks[0].X, blocks[0].Y), geom.V2(blocks[0].X, blocks[0].Y)
	for _, r := range blocks {
		lo, hi = geom.V2(min(lo.X, r.X), min(lo.Y, r.Y)), geom.V2(max(hi.X, r.X+r.W), max(hi.Y, r.Y+r.H))
	}
	box := boxOf(lo, hi)
	at := s.Position()
	st.Anchor = geom.V2(box.X+box.W/2-at.X, box.Y+box.H/2-at.Y).Scale(1 / k)
	st.Spread = geom.V2(box.W/2+baseFront, box.H/2+baseFront).Scale(1 / k)
}

// drawLit draws tile id as draw does at corner; one that blocks light,
// on a layer lit per pixel, takes its light just past the left, right and
// bottom of what it blocks with, standing on the cell at origin, never in
// its own shadow, never from behind it.
func (t *Tileset) drawLit(
	c *Canvas,
	id int,
	corner, origin, cell geom.Vec2,
	time float32,
	f TileFlip,
	tint color.Color,
	look *tileLook,
) {
	box, ok := t.face(id, cell, f)
	if !ok {
		t.draw(c, id, corner, cell, time, f, tint, look)
		return
	}
	saved := c.list.Light()
	c.list.SetLightFacing(geom.Rect{X: origin.X + box.X, Y: origin.Y + box.Y, W: box.W, H: box.H})
	t.draw(c, id, corner, cell, time, f, tint, look)
	c.list.RestoreLight(saved)
}

// faceKey is a tile as a map of some cell size draws it, flipped.
type faceKey struct {
	id   int
	cell geom.Vec2
	flip TileFlip
}

// blocksOf appends the rects tile id blocks light with, standing on the
// cell at origin, of size cell, flipped by f, as the light package reads
// them; none without it, when nothing is lit.
func (t *Tileset) blocksOf(dst []geom.Rect, id int, origin, cell geom.Vec2, f TileFlip) []geom.Rect {
	if lt.TileBlocks == nil {
		return dst
	}
	return lt.TileBlocks(t, id, origin, cell, uint8(f), dst)
}

// face is the box round what tile id blocks light with, grown by
// baseFront, from the corner of its cell, and whether it blocks any.
func (t *Tileset) face(id int, cell geom.Vec2, f TileFlip) (geom.Rect, bool) {
	if t.faceVersion != t.version {
		t.faceVersion, t.faces, t.faceList = t.version, nil, t.faceList[:0]
	}
	if f == 0 && cell == t.faceCell && id >= 0 && id < len(t.faceList) && t.faceList[id].known {
		e := t.faceList[id]
		return e.box, e.box.W > 0
	}
	box, ok := t.faceOf(id, cell, f)
	if f == 0 {
		if cell != t.faceCell {
			t.faceCell, t.faceList = cell, t.faceList[:0]
		}
		if id >= 0 && id < 1<<16 {
			for len(t.faceList) <= id {
				t.faceList = append(t.faceList, faceEntry{})
			}
			t.faceList[id] = faceEntry{box, true}
		}
	}
	return box, ok
}

// faceEntry is a tile's face box, known once worked out.
type faceEntry struct {
	box   geom.Rect
	known bool
}

// faceOf is face, through the map of every cell size and flip.
func (t *Tileset) faceOf(id int, cell geom.Vec2, f TileFlip) (geom.Rect, bool) {
	k := faceKey{id, cell, f}
	if b, ok := t.faces[k]; ok {
		return b, b.W > 0
	}
	var box geom.Rect
	for i, r := range t.blocksOf(nil, id, geom.Vec2{}, cell, f) {
		if i == 0 {
			box = r
			continue
		}
		box = boxOf(geom.V2(min(box.X, r.X), min(box.Y, r.Y)), geom.V2(max(box.X+box.W, r.X+r.W), max(box.Y+box.H, r.Y+r.H)))
	}
	if box.W > 0 {
		box = geom.Rect{X: box.X - baseFront, Y: box.Y - baseFront, W: box.W + 2*baseFront, H: box.H + 2*baseFront}
	}
	if t.faces == nil {
		t.faces = map[faceKey]geom.Rect{}
	}
	t.faces[k] = box
	return box, box.W > 0
}

// listLight sets how u's current list lights en, drawn through view, the
// list's points mapping to the light picture through toLight.
func listLight(u *Canvas, en entry, view, toLight geom.Transform2D) {
	u.list.SetShadedBy(draw.AllGroups &^ en.left)
	u.list.SetLightClass(u.classOf(en.named))
	switch en.light {
	case lt.PerPixel:
		u.list.SetLight(draw.LitPerPixel, toLight, geom.Vec2{})
	case lt.AtAnchor:
		if en.around.W > 0 || en.around.H > 0 {
			a := en.around
			u.list.SetLightAround(toLight, boxOf(view.Xform(geom.V2(a.X, a.Y)), view.Xform(geom.V2(a.X+a.W, a.Y+a.H))))
			return
		}
		u.list.SetLight(draw.LitAtAnchor, toLight, view.Xform(en.anchor2))
	case lt.Inherit, lt.Off, lt.Base:
		u.list.SetLight(draw.Unlit, toLight, geom.Vec2{})
	}
}

// screenTransform is e's transform to its window's points as drawn in
// the last frame, and the window: through the window's camera unless e is
// on a fixed layer.
func screenTransform(w *world.World, e Entity) (geom.Transform2D, *platform.Window, bool) {
	v := slot.Get[views](w)
	if v == nil {
		return geom.Transform2D{}, nil, false
	}
	root := e
	parallax, fixed, anchor := float32(1), e.GetDrawLook().overlay, geom.Vec2{}
	for p := e.GetSceneNode().ParentNode(); p != nil; p = p.ParentNode() {
		parent, ok := p.Entity().(Entity)
		if !ok {
			break
		}
		root = parent
		if l, isLayer := parent.(*Layer); isLayer && parallax == 1 && !fixed {
			parallax, fixed, anchor = l.parallax, l.fixed, l.anchor
		}
	}
	win := wm.Resolve(w, windowOf(w, root))
	t := drawnGlobal(e, v.alpha)
	if fixed {
		if win != nil {
			t.Origin = t.Origin.Add(anchor.Mul(win.Size()))
		}
		return t, win, true
	}
	cam := v.cameras[win]
	if cam == nil {
		return t, win, true
	}
	return cam.screen.Mul(cam.transform(cam.viewport, parallax)).Mul(t), win, true
}

// classOf is the light class of what is under named, a chain of layers
// some light Affects, in u's window this frame: 0 for none, else one for
// its innermost layer, given the first time it is asked for, and 0 past
// draw.MaxLightClasses.
func (u *Canvas) classOf(named *namedLayer) int {
	if named == nil {
		return 0
	}
	for i, c := range u.classes {
		if c[0] == named.layer {
			return i + 1
		}
	}
	if len(u.classes) >= draw.MaxLightClasses {
		return 0
	}
	var chain []any
	for n := named; n != nil; n = n.up {
		chain = append(chain, n.layer)
	}
	u.classes = append(u.classes, chain)
	return len(u.classes)
}
