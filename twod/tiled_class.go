package twod

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/lantervale/ember/color"
)

// ClassUse is what a Class may be the class of in Tiled: flags, so one
// class can serve several.
type ClassUse int

const (
	OnMap ClassUse = 1 << iota
	OnLayer
	OnObject
	OnTile
)

//nolint:gochecknoglobals // a lookup table
var classUses = []struct {
	use  ClassUse
	name string
}{{OnMap, "map"}, {OnLayer, "layer"}, {OnObject, typeObject}, {OnTile, useTile}}

// useTile is Tiled's word for a tile among a class's uses.
const useTile = "tile"

// Class is a custom class of a Tiled project, declared by the game: its
// name, what it is the class of, and its properties with their defaults.
// A map, layer, object or tile of the class has every one of them, the
// file's value where Tiled saved one, else the default: Tiled saves only
// what was changed. Shapes are the shapes an object of the class may
// have, any when empty. A tile's class adds to the tile properties every
// tile has (EngineTileProperties and TileProperties). Color is what Tiled
// draws its elements in, Tiled's own grey when zero.
type Class struct {
	Name       string
	On         ClassUse
	Shapes     []TiledShape
	Properties []Property
	Doc        string
	Color      color.Color
}

// Classes declares a game's Tiled classes. Once a class is declared for a
// use, an element of that use whose class isn't declared is an error, as is
// a property its class doesn't have, a value of the wrong kind or out of
// range, a Required property left empty and an object of a shape the class
// doesn't take; an element of no class stays as Tiled saved it. TiledProject
// writes the classes for Tiled.
func Classes(classes ...Class) TiledOption {
	return tiledOption(func(l *tiledLoad) { l.classes = append(l.classes, classes...) })
}

// property is the class's property called name; c may be nil, which has
// none.
func (c *Class) property(name string) (Property, bool) {
	if c == nil {
		return Property{}, false
	}
	for _, d := range c.Properties {
		if d.Name == name {
			return d, true
		}
	}
	return Property{}, false
}

// class is the declared class called name.
func (l *tiledLoad) class(name string) (*Class, bool) {
	if l == nil {
		return nil, false
	}
	for i := range l.classes {
		if l.classes[i].Name == name {
			return &l.classes[i], true
		}
	}
	return nil, false
}

// classOf is the declared class of an element of use whose class Tiled
// names name: nil for an element of no class, or of one when no class is
// declared for its use; an error for a class not declared for it.
func (l *tiledLoad) classOf(name string, use ClassUse) (*Class, error) {
	if name == "" {
		return nil, nil //nolint:nilnil // an element of no class has none
	}
	if name == maskClass || strings.HasPrefix(name, "ember.") {
		for _, c := range engineClasses() {
			if c.Name == name && c.On&use != 0 {
				return &c, nil
			}
		}
	}
	c, ok := l.class(name)
	switch {
	case ok && c.On&use != 0:
		return c, nil
	case ok:
		return nil, fmt.Errorf("class %s is not the class of a %s", name, useName(use))
	}
	var names []string
	for _, c := range l.classes {
		if c.On&use != 0 {
			names = append(names, c.Name)
		}
	}
	if len(names) == 0 {
		return nil, nil //nolint:nilnil // no class is declared for its use: it stays loose
	}
	return nil, fmt.Errorf("class %s is not declared%s", name, nearOf(name, names))
}

func useName(use ClassUse) string {
	for _, u := range classUses {
		if u.use == use {
			return u.name
		}
	}
	return "thing"
}

// nearOf says which of names name is likely a misspelling of.
func nearOf(name string, names []string) string {
	for _, n := range names {
		if edits(name, n) <= 2 {
			return fmt.Sprintf("; is it %s?", n)
		}
	}
	return ""
}

// propertiesOf are list's values for an element of use and class c, nil
// for one of no class: each checked against c's property, or the engine's
// for any element of use (elementProperties), a class-typed one's members
// filled, and every property of c the file leaves out at its default.
func (l *tiledLoad) propertiesOf(c *Class, use ClassUse, list []tiledProperty, dir string) (TiledProperties, error) {
	if c == nil {
		return l.loose(list, use, dir)
	}
	out := TiledProperties{}
	for _, p := range list {
		d, ok := c.property(p.Name)
		if !ok {
			d, ok = elementProperty(use, p.Name)
		}
		if !ok {
			return nil, fmt.Errorf("property %s: class %s has no such property%s", p.Name, c.Name, nearOf(p.Name,
				propertyNames(append(c.Properties, elementProperties(use)...))))
		}
		v, err := l.value(d, p.Type, p.PropertyType, p.Value, dir)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", p.Name, err)
		}
		out[p.Name] = v
	}
	if err := l.fill(c, out); err != nil {
		return nil, err
	}
	return out, nil
}

// loose are list's values as Tiled saved them, for an element of use and
// of no declared class; a property the engine reads on any element of use
// is still checked, and one of a declared class has its members checked
// and filled.
func (l *tiledLoad) loose(list []tiledProperty, use ClassUse, dir string) (TiledProperties, error) {
	out := properties(list, dir)
	for _, p := range list {
		if d, ok := elementProperty(use, p.Name); ok {
			v, err := l.value(d, p.Type, p.PropertyType, p.Value, dir)
			if err != nil {
				return nil, fmt.Errorf("property %s: %w", p.Name, err)
			}
			out[p.Name] = v
			continue
		}
		if p.Type != typeClass {
			continue
		}
		if _, ok := l.class(p.PropertyType); !ok {
			continue
		}
		v, err := l.value(Property{Kind: KindClass, Class: p.PropertyType}, p.Type, p.PropertyType, p.Value, dir)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", p.Name, err)
		}
		out[p.Name] = v
	}
	return out, nil
}

// fill sets every property of c out lacks to its default; a Required one
// it lacks, or holds empty, is an error.
func (l *tiledLoad) fill(c *Class, out TiledProperties) error {
	for _, d := range c.Properties {
		v, has := out[d.Name]
		if d.Required && (!has || v == "") {
			return fmt.Errorf("property %s: class %s requires it", d.Name, c.Name)
		}
		if has {
			continue
		}
		v, err := l.defaultOf(d)
		if err != nil {
			return fmt.Errorf("property %s: %w", d.Name, err)
		}
		out[d.Name] = v
	}
	return nil
}

// defaultOf is property d's default: a class-typed one's, its class's
// members at theirs.
func (l *tiledLoad) defaultOf(d Property) (any, error) {
	if d.Kind != KindClass {
		return d.Default, nil
	}
	return l.members(d.Class, nil, "")
}

// value is v, saved by Tiled as tiledType (of class propertyType for a
// class), read as d says, or what is wrong with it; a file's path is from
// dir.
func (l *tiledLoad) value(d Property, tiledType, propertyType string, v any, dir string) (any, error) {
	switch d.Kind {
	case KindClass:
		if tiledType != typeClass || propertyType != "" && propertyType != d.Class {
			return nil, fmt.Errorf("%v is a %s, not a %s", v, cmp.Or(propertyType, tiledType), d.Class)
		}
		m, _ := v.(map[string]any)
		return l.members(d.Class, m, dir)
	case KindFile:
		s, ok := v.(string)
		if !ok || tiledType != typeFile {
			return nil, fmt.Errorf("%v is a %s, not a file", v, tiledType)
		}
		if s == "" {
			return s, nil
		}
		return path.Join(dir, s), nil
	case KindObject:
		f, ok := v.(float64)
		if !ok || tiledType != typeObject {
			return nil, fmt.Errorf("%v is a %s, not an object", v, tiledType)
		}
		return int(f), nil
	case KindBool, KindInt, KindFloat, KindString, KindEnum, KindColor:
	}
	return d.check(tiledType, v)
}

// members are the members of class name Tiled saved as m, checked and
// filled with their defaults.
func (l *tiledLoad) members(name string, m map[string]any, dir string) (TiledProperties, error) {
	c, declared := l.class(name)
	if !declared {
		return nil, fmt.Errorf("class %s is not declared", name)
	}
	out := TiledProperties{}
	for key, v := range m {
		d, ok := c.property(key)
		if !ok {
			return nil, fmt.Errorf("member %s: class %s has no such property%s", key, name,
				nearOf(key, propertyNames(c.Properties)))
		}
		got, err := l.value(d, kindNames[d.Kind], d.Class, v, dir)
		if err != nil {
			return nil, fmt.Errorf("member %s: %w", key, err)
		}
		out[key] = got
	}
	if err := l.fill(c, out); err != nil {
		return nil, err
	}
	return out, nil
}

// propertyNamed is the property of defs called name.
func propertyNamed(defs []Property, name string) (Property, bool) {
	i := slices.IndexFunc(defs, func(d Property) bool { return d.Name == name })
	if i < 0 {
		return Property{}, false
	}
	return defs[i], true
}

func propertyNames(defs []Property) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}

// checkShape refuses an object whose shape class c doesn't take.
func checkShape(c *Class, shape TiledShape) error {
	if c == nil || len(c.Shapes) == 0 || slices.Contains(c.Shapes, shape) {
		return nil
	}
	names := make([]string, len(c.Shapes))
	for i, s := range c.Shapes {
		names[i] = shapeNames[s]
	}
	return fmt.Errorf("class %s is a %s, not a %s", c.Name, strings.Join(names, " or "), shapeNames[shape])
}

//nolint:gochecknoglobals // a lookup table
var shapeNames = [...]string{
	TiledRectangle: "rectangle",
	TiledEllipse:   "ellipse",
	TiledPoint:     "point",
	TiledPolygon:   "polygon",
	TiledPolyline:  "polyline",
	TiledText:      "text",
}

// tileValue is the default of property name for a tile of class name, when
// that class has the property.
func (l *tiledLoad) tileValue(class, name string) (any, bool) {
	c, ok := l.class(class)
	if !ok {
		return nil, false
	}
	d, ok := c.property(name)
	if !ok {
		return nil, false
	}
	v, err := l.defaultOf(d)
	return v, err == nil
}
