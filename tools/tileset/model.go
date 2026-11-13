package main

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// Kind is a shape's kind, as Tiled draws it.
type Kind int

const (
	Rect Kind = iota
	Ellipse
	Polygon
	Point
)

// Shape is one object of a tile's collision layer, in the tile's pixels:
// a rectangle or an ellipse in its box, or a polygon's points from X, Y.
// Shadow ones (class "shadow") block light only, Body ones (class "body")
// bodies only; the others both. from is the object as read, its other
// fields kept when it is written back.
type Shape struct {
	Kind   Kind
	X, Y   float32
	W, H   float32
	Points []geom.Vec2
	Shadow bool
	Body   bool
	from   *node
}

// class is the shape's Tiled class: "shadow", "body" or none.
func (s Shape) class() string {
	switch {
	case s.Shadow:
		return classShadow
	case s.Body:
		return classBody
	}
	return ""
}

// Bounds is the shape's box, in the tile's pixels.
func (s Shape) Bounds() geom.Rect {
	if s.Kind != Polygon || len(s.Points) == 0 {
		return geom.Rect{X: s.X, Y: s.Y, W: s.W, H: s.H}
	}
	lo, hi := s.Points[0], s.Points[0]
	for _, p := range s.Points {
		lo, hi = geom.V2(min(lo.X, p.X), min(lo.Y, p.Y)), geom.V2(max(hi.X, p.X), max(hi.Y, p.Y))
	}
	return geom.Rect{X: s.X + lo.X, Y: s.Y + lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}

// clone is s with its own points.
func (s Shape) clone() Shape {
	s.Points = slices.Clone(s.Points)
	return s
}

// Tile is what the tool edits of one tile: its shapes and the properties
// that say how it collides and casts light. Group is its occluder_group, -1
// for none.
type Tile struct {
	ID     int
	Shapes []Shape
	Solid  bool
	Tall   bool
	// SolidSet and TallSet say the file has solid and tall, true or
	// false; unset, the engine's default holds and nothing is written.
	SolidSet bool
	TallSet  bool
	NoShadow bool
	Group    int
	// Values are the tile's other declared properties it has, typed: a
	// bool, an int, a float64 or a string.
	Values map[string]any
}

func (t Tile) clone() Tile {
	shapes := make([]Shape, len(t.Shapes))
	for i, s := range t.Shapes {
		shapes[i] = s.clone()
	}
	t.Shapes = shapes
	t.Values = maps.Clone(t.Values)
	return t
}

// Tiled's words the tool reads and writes: the engine's property names and
// the shadow property's "none", shape classes, the inspector's "none" class,
// JSON keys, the raw true and the select tag.
const (
	propSolid         = "solid"
	propTall          = "tall"
	propShadow        = "shadow"
	propOccluderGroup = "occluder_group"
	propHeight        = "height"
	shadowNone        = "none"
	classShadow       = "shadow"
	classBody         = "body"
	noClass           = "none"
	keyType           = "type"
	keyHeight         = "height"
	keyPolygon        = "polygon"
	jsonTrue          = "true"
	tagSelect         = "select"
)

// core are the properties Tile has fields for.
//
//nolint:gochecknoglobals // a lookup table
var core = map[string]bool{propSolid: true, propTall: true, propShadow: true, propOccluderGroup: true}

// declared are the tile properties the tool knows: the engine's, and the
// game's when -props names a file of them.
//
//nolint:gochecknoglobals // set once from -props, before the tool starts
var declared = twod.EngineTileProperties()

// definition is property name's definition, when declared.
func definition(name string) (twod.Property, bool) {
	for _, d := range declared {
		if d.Name == name {
			return d, true
		}
	}
	return twod.Property{}, false
}

// typed is v read as d's kind.
func typed(d twod.Property, v *node) any {
	switch d.Kind {
	case twod.KindBool:
		return v.raw == jsonTrue
	case twod.KindInt, twod.KindObject:
		return int(v.num())
	case twod.KindFloat:
		return v.num()
	case twod.KindString, twod.KindEnum, twod.KindColor, twod.KindFile, twod.KindClass:
	}
	return v.str()
}

// valueNode is v as a JSON value.
func valueNode(v any) *node {
	switch x := v.(type) {
	case bool:
		return scalar(x)
	case int:
		return num(float32(x))
	case float64:
		return num(float32(x))
	}
	return scalar(fmt.Sprint(v))
}

// tiledType is the type Tiled saves a property of kind k as.
func tiledType(k twod.PropertyKind) string {
	switch k {
	case twod.KindBool:
		return "bool"
	case twod.KindInt:
		return "int"
	case twod.KindFloat:
		return "float"
	case twod.KindColor:
		return "color"
	case twod.KindFile:
		return "file"
	case twod.KindObject:
		return "object"
	case twod.KindClass:
		return "class"
	case twod.KindString, twod.KindEnum:
	}
	return "string"
}

// tileNode is the node of tile id in doc's tiles, nil without one.
func tileNode(doc *node, id int) *node {
	tiles := doc.get("tiles")
	if tiles == nil {
		return nil
	}
	for _, t := range tiles.items {
		if int(t.get("id").num()) == id {
			return t
		}
	}
	return nil
}

// tileIDs are the ids of doc's tiles: its tiles' entries for a collection,
// every cell of its grid for an atlas.
func tileIDs(doc *node) []int {
	var ids []int
	if doc.get("image") != nil {
		for id := range int(doc.get("tilecount").num()) {
			ids = append(ids, id)
		}
		return ids
	}
	if tiles := doc.get("tiles"); tiles != nil {
		for _, t := range tiles.items {
			ids = append(ids, int(t.get("id").num()))
		}
	}
	return ids
}

// readTile is tile id of doc.
func readTile(doc *node, id int) Tile {
	t := Tile{ID: id, Group: -1}
	n := tileNode(doc, id)
	if n == nil {
		return t
	}
	if props := n.get("properties"); props != nil {
		for _, p := range props.items {
			v := p.get("value")
			switch p.get("name").str() {
			case propSolid:
				t.Solid, t.SolidSet = v.raw == jsonTrue, true
			case propTall:
				t.Tall, t.TallSet = v.raw == jsonTrue, true
			case propShadow:
				t.NoShadow = v.str() == shadowNone
			case propOccluderGroup:
				t.Group = int(v.num())
			default:
				if d, ok := definition(p.get("name").str()); ok {
					if t.Values == nil {
						t.Values = map[string]any{}
					}
					t.Values[d.Name] = typed(d, v)
				}
			}
		}
	}
	if og := n.get("objectgroup"); og != nil && og.get("objects") != nil {
		for _, o := range og.get("objects").items {
			s := Shape{
				X:    float32(o.get("x").num()),
				Y:    float32(o.get("y").num()),
				W:    float32(o.get("width").num()),
				H:    float32(o.get(keyHeight).num()),
				from: o,
			}
			s.Shadow = o.get(keyType).str() == classShadow || o.get("class").str() == classShadow
			s.Body = o.get(keyType).str() == classBody || o.get("class").str() == classBody
			switch {
			case o.get("ellipse") != nil && o.get("ellipse").raw == jsonTrue:
				s.Kind = Ellipse
			case o.get("point") != nil && o.get("point").raw == jsonTrue:
				s.Kind = Point
			case o.get(keyPolygon) != nil:
				s.Kind = Polygon
				for _, p := range o.get(keyPolygon).items {
					s.Points = append(s.Points, geom.V2(float32(p.get("x").num()), float32(p.get("y").num())))
				}
			}
			t.Shapes = append(t.Shapes, s)
		}
	}
	return t
}

// writeTile puts t into doc: its shapes as its collision layer's objects,
// each keeping what it had but its place, size, class and points, and
// its properties, the others left as they were.
func writeTile(doc *node, t Tile) {
	n := tileNode(doc, t.ID)
	if n == nil {
		n = &node{object: true}
		n.set("id", num(float32(t.ID)))
		tiles := doc.get("tiles")
		if tiles == nil {
			tiles = &node{array: true}
			doc.set("tiles", tiles)
		}
		at := len(tiles.items)
		for i, other := range tiles.items {
			if int(other.get("id").num()) > t.ID {
				at = i
				break
			}
		}
		tiles.items = slices.Insert(tiles.items, at, n)
	}
	writeProperties(n, t)
	writeShapes(n, t.Shapes)
}

func writeProperties(n *node, t Tile) {
	props := n.get("properties")
	if props == nil {
		props = &node{array: true}
	}
	set := func(name, kind string, value *node, on bool) {
		at := -1
		for i, p := range props.items {
			if p.get("name").str() == name {
				at = i
			}
		}
		switch {
		case !on && at >= 0:
			props.items = slices.Delete(props.items, at, at+1)
		case on && at >= 0:
			props.items[at].set(keyType, scalar(kind))
			props.items[at].set("value", value)
		case on:
			p := &node{object: true}
			p.set("name", scalar(name))
			p.set(keyType, scalar(kind))
			p.set("value", value)
			props.items = append(props.items, p)
		}
	}
	set(propSolid, "bool", scalar(t.Solid), t.Solid || t.SolidSet)
	set(propTall, "bool", scalar(t.Tall), t.Tall || t.TallSet)
	set(propShadow, "string", scalar(shadowNone), t.NoShadow)
	set(propOccluderGroup, "int", num(float32(t.Group)), t.Group >= 0)
	for _, d := range declared {
		if core[d.Name] {
			continue
		}
		v, has := t.Values[d.Name]
		if !has {
			set(d.Name, tiledType(d.Kind), nil, false)
			continue
		}
		set(d.Name, tiledType(d.Kind), valueNode(v), true)
	}
	if len(props.items) == 0 {
		n.remove("properties")
		return
	}
	if n.get("properties") == nil {
		n.set("properties", props)
	}
}

func writeShapes(n *node, shapes []Shape) {
	if len(shapes) == 0 {
		n.remove("objectgroup")
		return
	}
	og := n.get("objectgroup")
	if og == nil {
		og = &node{object: true}
		for _, kv := range []struct {
			k string
			v *node
		}{{"draworder", scalar("index")}, {"id", num(2)}, {"name", scalar("")}, {"opacity", num(1)},
			{keyType, scalar("objectgroup")}, {"visible", scalar(true)}, {"x", num(0)}, {"y", num(0)}} {
			og.set(kv.k, kv.v)
		}
		n.set("objectgroup", og)
	}
	next := 1
	for _, s := range shapes {
		if s.from != nil {
			next = max(next, int(s.from.get("id").num())+1)
		}
	}
	objects := &node{array: true}
	for _, s := range shapes {
		o := s.from.clone()
		if o == nil {
			o = &node{object: true}
			o.set("id", num(float32(next)))
			next++
			o.set("name", scalar(""))
			o.set(keyType, scalar(""))
			o.set("rotation", num(0))
			o.set("visible", scalar(true))
		}
		class := s.class()
		if o.get("class") != nil {
			o.set("class", scalar(class))
		} else {
			o.set(keyType, scalar(class))
		}
		o.set("x", num(s.X))
		o.set("y", num(s.Y))
		switch s.Kind {
		case Polygon:
			o.set("width", num(0))
			o.set(keyHeight, num(0))
			pts := &node{array: true}
			for _, p := range s.Points {
				pn := &node{object: true}
				pn.set("x", num(p.X))
				pn.set("y", num(p.Y))
				pts.items = append(pts.items, pn)
			}
			o.set(keyPolygon, pts)
		case Point:
		case Rect, Ellipse:
			o.set("width", num(s.W))
			o.set(keyHeight, num(s.H))
			if s.Kind == Ellipse {
				o.set("ellipse", scalar(true))
			} else {
				o.remove("ellipse")
			}
			o.remove(keyPolygon)
		}
		objects.items = append(objects.items, o)
	}
	og.set("objects", objects)
}

// num is v as a JSON number, written as Tiled writes it: whole numbers
// without a point.
func num(v float32) *node {
	if v == float32(int(v)) {
		return &node{raw: strconv.Itoa(int(v))}
	}
	return &node{raw: fmt.Sprint(v)}
}
