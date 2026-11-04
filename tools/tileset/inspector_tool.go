package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The Tool's side of the inspector: what it shows of the editor, and what
// it does when a view asks.

// objectIDs are the ids the tile's objects have, or will have once
// written: a new one takes the next after the highest, as writeShapes
// gives it.
func objectIDs(shapes []Shape) []int {
	next := 1
	for _, s := range shapes {
		if s.from != nil {
			next = max(next, int(s.from.get("id").num())+1)
		}
	}
	ids := make([]int, len(shapes))
	for i, s := range shapes {
		if s.from != nil {
			ids[i] = int(s.from.get("id").num())
		} else {
			ids[i] = next
			next++
		}
	}
	return ids
}

// typeOf is the shape's type as the file writes it: none, shadow or body.
func typeOf(s Shape) string {
	if c := s.class(); c != "" {
		return c
	}
	return noClass
}

// objectChose applies a type or shape picked for an object.
func (t *Tool) objectChose(at objectField, index int) {
	ed := t.ed
	if at.i >= len(ed.tile.Shapes) || index < 0 {
		return
	}
	ed.selected, ed.point = at.i, -1
	switch at.key {
	case keyType:
		ed.SetClass([]string{"", classShadow, classBody}[index])
	case "shape":
		ed.SetKind(Kind(index))
	}
}

// objectTyped applies a value typed into an object's field: x, y, width,
// height, or a polygon's points as "x,y x,y …".
func (t *Tool) objectTyped(at objectField, text string) {
	ed := t.ed
	if at.i >= len(ed.tile.Shapes) {
		return
	}
	ed.selected, ed.point = at.i, -1
	s := ed.tile.Shapes[at.i]
	text = strings.TrimSpace(text)
	if at.key == "points" {
		var pts []geom.Vec2
		for _, pair := range strings.Fields(text) {
			x, y, ok := strings.Cut(pair, ",")
			px, errX := strconv.ParseFloat(x, 32)
			py, errY := strconv.ParseFloat(y, 32)
			if !ok || errX != nil || errY != nil {
				t.status.SetText("points: x,y pairs split by spaces")
				return
			}
			pts = append(pts, geom.V2(float32(px), float32(py)))
		}
		if err := ed.SetPoints(pts); err != nil {
			t.status.SetText(err.Error())
		}
		return
	}
	n, err := strconv.ParseFloat(text, 32)
	if err != nil {
		t.status.SetText(at.key + ": not a number")
		return
	}
	v := float32(n)
	x, y, w, h := s.X, s.Y, s.W, s.H
	switch at.key {
	case "x":
		x = v
	case "y":
		y = v
	case "width":
		w = v
	case keyHeight:
		h = v
	}
	ed.SetObject(x, y, w, h)
}

// animationText is the tile's animation as the file has it: each frame's
// tile id and duration in milliseconds, or none.
func animationText(doc *node, id int) string {
	n := tileNode(doc, id)
	if n == nil || n.get("animation") == nil || len(n.get("animation").items) == 0 {
		return "none"
	}
	var frames []string
	for _, f := range n.get("animation").items {
		frames = append(frames, fmt.Sprintf("%d · %d ms", int(f.get("tileid").num()), int(f.get("duration").num())))
	}
	return strings.Join(frames, ", ")
}

// nearMisses are tile id's properties named almost as one the engine
// reads, each with the name it likely meant: a typo the engine would
// silently ignore.
func nearMisses(doc *node, id int) []string {
	n := tileNode(doc, id)
	if n == nil || n.get("properties") == nil {
		return nil
	}
	var out []string
	for _, p := range n.get("properties").items {
		name := p.get("name").str()
		if _, ok := definition(name); ok {
			continue
		}
		if near, ok := twod.NearMiss(name, engineProperties()...); ok {
			v := p.get("value")
			text := v.raw
			if s := v.str(); s != "" {
				text = s
			}
			out = append(out, fmt.Sprintf("%s = %s  (%s?)", name, text, near))
		}
	}
	return out
}

// inspect shows the inspector the picked tile, or the tiles marked, when
// it, its version, the selection or the marks changed.
func (t *Tool) inspect() {
	ed := t.ed
	key := fmt.Sprint(ed.path, ed.version, ed.tile.ID, ed.selected, t.marks(), t.err != nil)
	if key == t.inspected {
		return
	}
	t.inspected = key
	t.view.Show(t.inspectorState())
}

// inspectorState is what the inspector shows of the editor now.
func (t *Tool) inspectorState() inspectorState {
	ed := t.ed
	tile := ed.tile
	s := inspectorState{Title: tileTitle(ed.doc, tile.ID), Loaded: ed.path != "", Selected: -1,
		Animation: animationText(ed.doc, tile.ID), NearMisses: nearMisses(ed.doc, tile.ID)}
	tiles := ed.targets()
	if len(tiles) > 1 {
		s.Title = fmt.Sprintf("%d tiles", len(tiles))
	}
	differ := func(name string) bool {
		for _, other := range tiles[1:] {
			if propertyOf(other, name) != propertyOf(tiles[0], name) {
				return true
			}
		}
		return false
	}
	for _, d := range engineProperties() {
		p := propertyState{
			Name:    d.Name,
			Kind:    d.Kind,
			Default: fmt.Sprint(d.Default),
			Values:  valuesOf(d),
			Mixed:   differ(d.Name),
		}
		switch {
		case d.Kind == twod.KindBool || p.Values != nil:
			p.Value, p.Set = shownValue(tile, d)
		default:
			if v, ok := tile.Values[d.Name]; ok {
				p.Value, p.Set = fmt.Sprint(v), true
			}
			p.Hint = placeholder(d)
			if d.Name == propHeight {
				p.Hint = t.autoHeight()
			}
		}
		s.Properties = append(s.Properties, p)
	}
	ids := objectIDs(tile.Shapes)
	for i, sh := range tile.Shapes {
		o := objectState{
			ID:     ids[i],
			Type:   typeOf(sh),
			Shape:  objectKinds[sh.Kind],
			X:      sh.X,
			Y:      sh.Y,
			W:      sh.W,
			H:      sh.H,
			Points: sh.Points,
		}
		if sh.Kind == Polygon {
			o.W, o.H = 0, 0
		}
		s.Objects = append(s.Objects, o)
	}
	if ed.selected >= 0 && ed.selected < len(tile.Shapes) {
		s.Selected = ed.selected
	}
	return s
}

// tileTitle is "Tile <id>", with its class when Tiled gave it one: the
// class field, or type in files from before Tiled 1.9.
func tileTitle(doc *node, id int) string {
	title := fmt.Sprintf("Tile %d", id)
	n := tileNode(doc, id)
	if n == nil {
		return title
	}
	class := n.get("class").str()
	if class == "" {
		class = n.get(keyType).str()
	}
	if class != "" {
		title += " · " + class
	}
	return title
}

// SetBool, SetChoice and the rest are what the Tool does when the
// inspector asks (inspectorIntents).
func (t *Tool) SetBool(name string, on bool)        { t.ed.SetValue(name, on) }
func (t *Tool) SetChoice(name string, index int)    { t.chose(name, index) }
func (t *Tool) SetField(name, text string)          { t.property(name, text) }
func (t *Tool) ObjectField(i int, key, text string) { t.objectTyped(objectField{i, key}, text) }
func (t *Tool) DeleteObject()                       { t.deleteSelected() }

func (t *Tool) SelectObject(i int) {
	if i >= 0 && i < len(t.ed.tile.Shapes) {
		t.ed.selected, t.ed.point = i, -1
	}
}

func (t *Tool) ObjectChoice(i int, key string, index int) {
	t.objectChose(objectField{i, key}, index)
}

func (t *Tool) AddObject(kind Kind) {
	if t.ed.path != "" {
		t.ed.Add(kind, false)
	}
}
