package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
)

// Editor is the tool's state: the tileset file, its document, the tile
// picked and its shape selected, and the changes that undo and redo walk.
type Editor struct {
	path    string
	doc     *node
	newline bool
	// read is the file as it was read or last saved, to refuse saving
	// over a change made elsewhere.
	read     []byte
	ids      []int
	tile     Tile
	selected int
	undo     []change
	redo     []change
	// version counts the changes, for what is drawn from the document.
	version int
	saved   int
	// size is the picked tile's picture size, in its pixels.
	size geom.Vec2
	// free lets shapes off whole art pixels, to quarters of them.
	free bool
	// point is the selected polygon's selected point, -1 for none.
	point int
	// marked are the tiles picked with the one picked, by Shift or
	// Command, whose properties and shapes edits reach too.
	marked map[int]bool
	// clip are the shapes Copy kept.
	clip []Shape
}

// change is one edit of one tile: as it was, and as it became.
// change is one edit of one or more tiles: as they were, and as they
// became, the picked tile first.
type change struct{ before, after []Tile }

// open reads the tileset at path.
func open(path string) (*Editor, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := parse(b)
	if err != nil {
		return nil, err
	}
	e := &Editor{
		path:     path,
		doc:      doc,
		newline:  bytes.HasSuffix(b, []byte("\n")),
		read:     b,
		ids:      tileIDs(doc),
		selected: -1,
		point:    -1,
	}
	if len(e.ids) > 0 {
		e.pick(e.ids[0])
	}
	return e, nil
}

// pick makes tile id the one edited.
func (e *Editor) pick(id int) {
	e.tile, e.selected, e.point = readTile(e.doc, id), -1, -1
	e.size = tileSize(e.doc, id)
}

// tileSize is tile id's picture size in pixels: its own for a collection,
// the grid's for an atlas.
func tileSize(doc *node, id int) geom.Vec2 {
	if n := tileNode(doc, id); n != nil && n.get("width") != nil {
		return geom.V2(float32(n.get("width").num()), float32(n.get(keyHeight).num()))
	}
	if n := tileNode(doc, id); n != nil && n.get("imagewidth") != nil && doc.get("image") == nil {
		return geom.V2(float32(n.get("imagewidth").num()), float32(n.get("imageheight").num()))
	}
	return geom.V2(float32(doc.get("tilewidth").num()), float32(doc.get("tileheight").num()))
}

// apply makes after the picked tile, as one change undo takes back.
func (e *Editor) apply(after Tile) { e.applyAll([]Tile{after}) }

// applyAll makes after the tiles they are, as one change undo takes
// back.
func (e *Editor) applyAll(after []Tile) {
	c := change{}
	for _, t := range after {
		before := readTile(e.doc, t.ID)
		if t.ID == e.tile.ID {
			before = e.tile.clone()
		}
		c.before, c.after = append(c.before, before), append(c.after, t.clone())
	}
	e.undo = append(e.undo, c)
	e.redo = e.redo[:0]
	e.setAll(after)
}

// setAll writes tiles into the document, the picked one shown as it is.
func (e *Editor) setAll(tiles []Tile) {
	for _, t := range tiles {
		if t.ID == e.tile.ID {
			e.set(t)
			continue
		}
		writeTile(e.doc, t.clone())
		e.version++
	}
}

func (e *Editor) set(t Tile) {
	e.tile = t.clone()
	writeTile(e.doc, e.tile)
	e.version++
	if e.selected >= len(e.tile.Shapes) {
		e.selected = len(e.tile.Shapes) - 1
	}
}

// Undo takes the last change back, picking its tile.
func (e *Editor) Undo() {
	if len(e.undo) == 0 {
		return
	}
	c := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.redo = append(e.redo, c)
	e.keep(c.before)
}

// targets are the tiles an edit of properties or shapes goes to: the one
// picked and the ones marked with it.
func (e *Editor) targets() []Tile {
	out := []Tile{e.tile.clone()}
	for _, id := range e.ids {
		if e.marked[id] && id != e.tile.ID {
			out = append(out, readTile(e.doc, id))
		}
	}
	return out
}

// SetFlag sets property name, solid, tall or noShadow, on every target.
func (e *Editor) SetFlag(name string, on bool) {
	ts := e.targets()
	for i := range ts {
		switch name {
		case propSolid:
			ts[i].Solid = on
		case propTall:
			ts[i].Tall = on
		case "no-shadow":
			ts[i].NoShadow = on
		}
	}
	e.applyAll(ts)
}

// SetValue sets declared property name to v on every target, removing it
// when v is its default; the core ones set their fields.
func (e *Editor) SetValue(name string, v any) {
	d, ok := definition(name)
	if !ok {
		return
	}
	ts := e.targets()
	for i := range ts {
		t := &ts[i]
		switch name {
		case propSolid:
			t.Solid, _ = v.(bool)
			t.SolidSet = true
		case propTall:
			t.Tall, _ = v.(bool)
			t.TallSet = true
		case propShadow:
			t.NoShadow = v == shadowNone
		case propOccluderGroup:
			if g, isInt := v.(int); isInt {
				t.Group = g
			}
		default:
			if t.Values == nil {
				t.Values = map[string]any{}
			}
			if v == d.Default {
				delete(t.Values, name)
			} else {
				t.Values[name] = v
			}
		}
	}
	e.applyAll(ts)
}

// SetGroup sets the occluder group, -1 for none, on every target.
func (e *Editor) SetGroup(g int) {
	ts := e.targets()
	for i := range ts {
		ts[i].Group = g
	}
	e.applyAll(ts)
}

// Copy keeps the selected shape, or every shape of the picked tile when
// none is selected, for Paste.
func (e *Editor) Copy() int {
	e.clip = e.clip[:0]
	for i, s := range e.tile.Shapes {
		if e.selected < 0 || i == e.selected {
			s = s.clone()
			s.from = nil
			e.clip = append(e.clip, s)
		}
	}
	return len(e.clip)
}

// Paste adds the shapes copied to the picked tile, kept in it, and
// selects the last.
func (e *Editor) Paste() {
	if len(e.clip) == 0 {
		return
	}
	t := e.tile.clone()
	for _, s := range e.clip {
		t.Shapes = append(t.Shapes, e.clamp(s.clone(), e.room(s)))
	}
	e.apply(t)
	e.selected, e.point = len(t.Shapes)-1, -1
}

// ApplyShapes gives every tile marked the picked tile's shapes.
func (e *Editor) ApplyShapes() int {
	ts := e.targets()[1:]
	for i := range ts {
		ts[i].Shapes = ts[i].Shapes[:0]
		for _, s := range e.tile.Shapes {
			s = s.clone()
			s.from = nil
			ts[i].Shapes = append(ts[i].Shapes, s)
		}
	}
	if len(ts) > 0 {
		e.applyAll(ts)
	}
	return len(ts)
}

// Redo makes the last change taken back again.
func (e *Editor) Redo() {
	if len(e.redo) == 0 {
		return
	}
	c := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.undo = append(e.undo, c)
	e.keep(c.after)
}

// keep makes tiles as they were, picking the first unless the picked one
// is among them, its selection then kept.
func (e *Editor) keep(tiles []Tile) {
	sel, point := e.selected, e.point
	same := slices.ContainsFunc(tiles, func(t Tile) bool { return t.ID == e.tile.ID })
	if !same {
		e.pick(tiles[0].ID)
	}
	marked := e.marked
	e.setAll(tiles)
	e.marked = marked
	t := e.tile
	if same && sel < len(t.Shapes) {
		e.selected, e.point = sel, point
		if s, ok := e.polygon(); !ok || point >= len(s.Points) {
			e.point = -1
		}
	}
}

// Changed reports whether there are changes not saved.
func (e *Editor) Changed() bool { return e.version != e.saved }

// extent is the picked tile and its shapes, in its pixels.
func (e *Editor) extent() geom.Rect {
	x0, y0, x1, y1 := float32(0), float32(0), e.size.X, e.size.Y
	for _, s := range e.tile.Shapes {
		b := s.Bounds()
		x0, y0, x1, y1 = min(x0, b.X), min(y0, b.Y), max(x1, b.X+b.W), max(y1, b.Y+b.H)
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// room is where a shape that was at before may go: the picked tile, and
// as far past it as before already reached (a wall's box reaching under
// its picture).
func (e *Editor) room(before Shape) geom.Rect {
	b := before.Bounds()
	x0, y0 := min(0, b.X), min(0, b.Y)
	x1, y1 := max(e.size.X, b.X+b.W), max(e.size.Y, b.Y+b.H)
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// clamp keeps s inside r: moved back in, and no larger than it.
func (e *Editor) clamp(s Shape, r geom.Rect) Shape {
	s = s.clone()
	in := func(v, lo, hi float32) float32 { return min(max(v, lo), max(hi, lo)) }
	switch s.Kind {
	case Point:
		s.X, s.Y = in(s.X, r.X, r.X+r.W), in(s.Y, r.Y, r.Y+r.H)
	case Polygon:
		for i, p := range s.Points {
			s.Points[i] = geom.V2(in(s.X+p.X, r.X, r.X+r.W)-s.X, in(s.Y+p.Y, r.Y, r.Y+r.H)-s.Y)
		}
	case Rect, Ellipse:
		s.W, s.H = in(s.W, 1, r.W), in(s.H, 1, r.H)
		s.X, s.Y = in(s.X, r.X, r.X+r.W-s.W), in(s.Y, r.Y, r.Y+r.H-s.H)
	}
	return s
}

// inside is polygon s moved whole back into r.
func (e *Editor) inside(s Shape, r geom.Rect) Shape {
	b := s.Bounds()
	s.X -= max(b.X+b.W-(r.X+r.W), 0) + min(b.X-r.X, 0)
	s.Y -= max(b.Y+b.H-(r.Y+r.H), 0) + min(b.Y-r.Y, 0)
	return s
}

// snap rounds v to a whole art pixel.
func snap(v float32) float32 { return float32(math.Round(float64(v))) }

// fit is v snapped to a whole art pixel while the editor snaps, else to a
// quarter of one.
func (e *Editor) fit(v float32) float32 {
	if e.free {
		return float32(math.Round(float64(v)*4) / 4)
	}
	return snap(v)
}

// Add adds a shape of kind to the picked tile, a quarter of it in its
// middle, light-only when shadow, and selects it.
func (e *Editor) Add(kind Kind, shadow bool) {
	t := e.tile.clone()
	w, h := snap(e.size.X/2), snap(e.size.Y/2)
	step := float32(len(t.Shapes)%4) * 2
	s := Shape{Kind: kind, X: snap((e.size.X-w)/2) + step, Y: snap((e.size.Y-h)/2) + step, W: w, H: h, Shadow: shadow}
	if kind == Polygon {
		s.W, s.H = 0, 0
		s.Points = []geom.Vec2{{}, {X: w}, {X: w / 2, Y: h}}
		s = e.inside(s, geom.Rect{W: e.size.X, H: e.size.Y})
	}
	t.Shapes = append(t.Shapes, e.clamp(s, geom.Rect{W: e.size.X, H: e.size.Y}))
	e.apply(t)
	e.selected = len(t.Shapes) - 1
}

// errFewPoints is DeletePoint's answer for a polygon of three points.
var errFewPoints = errors.New("a polygon keeps at least three points")

// polygon is the selected shape when it is a polygon.
func (e *Editor) polygon() (Shape, bool) {
	if e.selected < 0 || e.selected >= len(e.tile.Shapes) || e.tile.Shapes[e.selected].Kind != Polygon {
		return Shape{}, false
	}
	return e.tile.Shapes[e.selected], true
}

// InsertPoint puts a point into the selected polygon after its point i, at
// at in the tile's pixels, snapped and kept in the tile, and selects it.
func (e *Editor) InsertPoint(i int, at geom.Vec2) {
	s, ok := e.polygon()
	if !ok || i < 0 || i >= len(s.Points) {
		return
	}
	t := e.tile.clone()
	p := &t.Shapes[e.selected]
	room := e.room(*p)
	at = geom.V2(min(max(e.fit(at.X), room.X), room.X+room.W), min(max(e.fit(at.Y), room.Y), room.Y+room.H))
	p.Points = slices.Insert(p.Points, i+1, at.Sub(geom.V2(p.X, p.Y)))
	e.apply(t)
	e.point = i + 1
}

// DeletePoint takes the selected point out of the selected polygon, which
// keeps at least three.
func (e *Editor) DeletePoint() error {
	s, ok := e.polygon()
	if !ok || e.point < 0 || e.point >= len(s.Points) {
		return nil
	}
	if len(s.Points) <= 3 {
		return errFewPoints
	}
	t := e.tile.clone()
	p := &t.Shapes[e.selected]
	p.Points = slices.Delete(p.Points, e.point, e.point+1)
	i := e.point
	e.apply(t)
	e.point = min(i, len(p.Points)-1)
	return nil
}

// Delete deletes the selected point of a polygon when one is, else the
// selected shape.
func (e *Editor) Delete() {
	if e.selected < 0 {
		return
	}
	if _, ok := e.polygon(); ok && e.point >= 0 {
		_ = e.DeletePoint()
		return
	}
	t := e.tile.clone()
	i := e.selected
	t.Shapes = append(t.Shapes[:i], t.Shapes[i+1:]...)
	e.apply(t)
	e.selected, e.point = min(i, len(t.Shapes)-1), -1
}

// SetClass makes the selected shape collision (""), light only ("shadow")
// or body only ("body").
func (e *Editor) SetClass(class string) {
	if e.selected < 0 {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	if s.class() == class {
		return
	}
	s.Shadow, s.Body = class == classShadow, class == classBody
	e.apply(t)
}

// ToggleShadow steps the selected shape from collision to light only, to
// body only, and back to collision.
func (e *Editor) ToggleShadow() {
	if e.selected < 0 {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	switch {
	case s.Shadow:
		s.Shadow, s.Body = false, true
	case s.Body:
		s.Body = false
	default:
		s.Shadow = true
	}
	e.apply(t)
}

// Move moves the selected shape by d art pixels, whole ones.
func (e *Editor) Move(d geom.Vec2) {
	if e.selected < 0 || d == (geom.Vec2{}) {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	room := e.room(*s)
	s.X, s.Y = e.fit(s.X+d.X), e.fit(s.Y+d.Y)
	if s.Kind == Polygon {
		*s = e.inside(*s, room)
	}
	*s = e.clamp(*s, room)
	e.apply(t)
}

// Resize sets the selected rectangle's or ellipse's box, snapped to art
// pixels, at least one pixel wide and high.
func (e *Editor) Resize(r geom.Rect) {
	if e.selected < 0 || e.tile.Shapes[e.selected].Kind == Polygon || e.tile.Shapes[e.selected].Kind == Point {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	room := e.room(*s)
	s.X, s.Y = e.fit(r.X), e.fit(r.Y)
	s.W, s.H = max(e.fit(r.W), 1), max(e.fit(r.H), 1)
	*s = e.clamp(*s, room)
	e.apply(t)
}

// MovePoint moves point i of the selected polygon to at, in the tile's
// pixels, snapped.
func (e *Editor) MovePoint(i int, at geom.Vec2) {
	if e.selected < 0 || e.tile.Shapes[e.selected].Kind != Polygon {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	if i < 0 || i >= len(s.Points) {
		return
	}
	room := e.room(*s)
	s.Points[i] = geom.V2(e.fit(at.X-s.X), e.fit(at.Y-s.Y))
	*s = e.clamp(*s, room)
	e.apply(t)
}

// SetBox places the selected shape at x, y and, a rectangle or an
// ellipse, sizes it w by h, in the tile's pixels.
func (e *Editor) SetBox(x, y, w, h float32) {
	if e.selected < 0 {
		return
	}
	s := e.tile.Shapes[e.selected]
	switch s.Kind {
	case Rect, Ellipse:
		e.Resize(geom.Rect{X: x, Y: y, W: w, H: h})
	case Polygon, Point:
		b := s.Bounds()
		e.Move(geom.V2(x-b.X, y-b.Y))
	}
}

// SetProperties changes the picked tile's properties.
func (e *Editor) SetProperties(solid, tall, noShadow bool, group int) {
	t := e.tile.clone()
	t.Solid, t.Tall, t.NoShadow, t.Group = solid, tall, noShadow, group
	e.apply(t)
}

// errChangedOnDisk is Save's answer when the file changed since it was
// read.
var errChangedOnDisk = errors.New("the file changed on disk since it was opened: not saved")

// Save writes the document back to its file, unless the file changed
// since it was read or last saved.
func (e *Editor) Save() error {
	now, err := os.ReadFile(e.path)
	if err != nil {
		return err
	}
	if !bytes.Equal(now, e.read) {
		return errChangedOnDisk
	}
	b := e.doc.bytes(e.newline)
	//nolint:gosec // the user's own tileset, readable like the rest of their project
	if err = os.WriteFile(e.path, b, 0o644); err != nil {
		return err
	}
	e.read, e.saved = b, e.version
	return nil
}

// SaveAs writes the document to path, which becomes the editor's file,
// its pictures' paths made relative to path's folder; it never replaces
// another file.
func (e *Editor) SaveAs(path string) error {
	if _, err := os.Stat(path); err == nil && path != e.path {
		return fmt.Errorf("%s already exists: pick another name", filepath.Base(path))
	}
	from, to := filepath.Dir(e.path), filepath.Dir(path)
	if from != to {
		rebase := func(n *node) {
			if img := n.get("image"); img != nil {
				if rel, err := filepath.Rel(to, filepath.Join(from, img.str())); err == nil {
					n.set("image", scalar(filepath.ToSlash(rel)))
				}
			}
		}
		rebase(e.doc)
		if tiles := e.doc.get("tiles"); tiles != nil {
			for _, t := range tiles.items {
				rebase(t)
			}
		}
	}
	b := e.doc.bytes(e.newline)
	//nolint:gosec // the user's own tileset, readable like the rest of their project
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	e.path, e.read, e.saved = path, b, e.version
	return nil
}

// blank is an editor of no tileset, for the tool before one is opened.
func blank() *Editor {
	return &Editor{doc: &node{object: true}, selected: -1, point: -1, tile: Tile{Group: -1}}
}

// Overwrite writes the document over its file whatever happened to it on
// disk since it was read.
func (e *Editor) Overwrite() error {
	b := e.doc.bytes(e.newline)
	//nolint:gosec // the user's own tileset, readable like the rest of their project
	if err := os.WriteFile(e.path, b, 0o644); err != nil {
		return err
	}
	e.read, e.saved = b, e.version
	return nil
}

// fs is the tileset's folder, its file as edited so far: what the
// preview loads through the engine's own reader.
// It is rooted at the file system's root, name being the file's path from
// it, so pictures in a folder above the file's are found too.
func (e *Editor) fs() (fs.FS, string) {
	abs, err := filepath.Abs(e.path)
	if err != nil {
		abs = e.path
	}
	root := filepath.VolumeName(abs) + string(filepath.Separator)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		rel = abs
	}
	name := filepath.ToSlash(rel)
	return overlay{dir: os.DirFS(root), name: name, data: e.doc.bytes(e.newline)}, name
}

type overlay struct {
	dir  fs.FS
	name string
	data []byte
}

func (o overlay) Open(name string) (fs.File, error) {
	if name == o.name {
		return fstest.MapFS{name: {Data: o.data}}.Open(name)
	}
	return o.dir.Open(name)
}

// SetKind makes the selected object a rectangle, an ellipse or a polygon:
// a polygon takes its box's corners as points, a box its polygon's bounds.
func (e *Editor) SetKind(k Kind) {
	if e.selected < 0 || k == Point || e.tile.Shapes[e.selected].Kind == k {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	switch {
	case k == Polygon:
		w, h := max(s.W, 1), max(s.H, 1)
		s.Points = []geom.Vec2{{}, geom.V2(w, 0), geom.V2(w, h), geom.V2(0, h)}
		s.W, s.H = 0, 0
	case s.Kind == Polygon:
		b := s.Bounds()
		s.X, s.Y, s.W, s.H, s.Points = b.X, b.Y, max(b.W, 1), max(b.H, 1), nil
	}
	s.Kind = k
	e.apply(t)
}

// SetPoints sets the selected polygon's points, from its x, y, keeping
// three at least.
func (e *Editor) SetPoints(pts []geom.Vec2) error {
	if e.selected < 0 || e.tile.Shapes[e.selected].Kind != Polygon {
		return nil
	}
	if len(pts) < 3 {
		return errors.New("a polygon keeps three points at least")
	}
	t := e.tile.clone()
	t.Shapes[e.selected].Points = slices.Clone(pts)
	e.apply(t)
	return nil
}

// SetObject sets the selected object's x, y and, a rectangle or an
// ellipse, its width and height, as typed: a polygon moves its points
// with its x, y.
func (e *Editor) SetObject(x, y, w, h float32) {
	if e.selected < 0 {
		return
	}
	t := e.tile.clone()
	s := &t.Shapes[e.selected]
	s.X, s.Y = x, y
	if s.Kind == Rect || s.Kind == Ellipse {
		s.W, s.H = max(w, 0), max(h, 0)
	}
	e.apply(t)
}
