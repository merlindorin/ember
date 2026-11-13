package twod

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/merlindorin/ember/color"
)

// PropertyKind is the type of a property: what Tiled shows and the loader
// accepts.
type PropertyKind int

const (
	KindBool PropertyKind = iota
	KindInt
	KindFloat
	KindString
	// KindEnum is a string out of a Property's Values.
	KindEnum
	KindColor
	// KindClass is the members of the declared class a Property's Class
	// names, as TiledProperties.
	KindClass
	// KindFile is a file's path from the file system's root.
	KindFile
	// KindObject is an object's id.
	KindObject
)

// The names Tiled spells its property types with, and the engine's tile
// properties some code reads by name.
const (
	typeBool   = "bool"
	typeInt    = "int"
	typeFloat  = "float"
	typeString = "string"
	typeClass  = "class"
	typeFile   = "file"
	typeObject = "object"
	propShadow = "shadow"
	propHeight = "height"
)

//nolint:gochecknoglobals // a lookup table
var kindNames = [...]string{
	KindBool:   typeBool,
	KindInt:    typeInt,
	KindFloat:  typeFloat,
	KindString: typeString,
	KindEnum:   typeString,
	KindColor:  "color",
	KindClass:  typeClass,
	KindFile:   typeFile,
	KindObject: typeObject,
}

// Property defines a property of a tile or of a Class: its name, kind,
// default and, for an enum, its values, for a number its range when
// Min < Max, for KindClass the declared class its members are of; Doc says
// what it does. Required refuses it left out or empty in a class's
// element. The engine's tile properties are EngineTileProperties; a game
// adds its own with TileProperties, and its classes' with Classes.
type Property struct {
	Name     string
	Kind     PropertyKind
	Default  any
	Values   []string
	Min, Max float64
	Required bool
	Class    string
	Doc      string
}

// EngineTileProperties are the tile properties the engine reads.
func EngineTileProperties() []Property {
	return []Property{
		{
			Name:    "solid",
			Kind:    KindBool,
			Default: false,
			Doc:     "blocks bodies, as its collision shapes or its whole cell; true when it has shapes",
		},
		{
			Name:    "tall",
			Kind:    KindBool,
			Default: false,
			Doc:     "drawn from its foot, sorted by y with characters, lit at its base",
		},
		{
			Name:    propShadow,
			Kind:    KindEnum,
			Default: "cast",
			Values:  []string{"cast", "none"},
			Doc:     "none: blocks bodies, not light",
		},
		{Name: "occluder_group", Kind: KindInt, Default: 0, Min: 0, Max: 2, Doc: "occluder group, for BlockedBy"},
		{
			Name:    propHeight,
			Kind:    KindFloat,
			Default: 0.0,
			Min:     0,
			Max:     4096,
			Doc: "how tall it stands, in its pixels, for shadow length; " +
				"0: its picture's height when tall, its cell's when solid",
		},
		{
			Name:    "emissive",
			Kind:    KindFloat,
			Default: 0.0,
			Min:     0,
			Max:     4096,
			Doc:     "glows over this reach, in its pixels",
		},
	}
}

// TiledOption sets up the loading of a Tiled tileset or map.
type TiledOption interface {
	TileMapOption
	tiled(*tiledLoad)
}

// tiledLoad is what loading a Tiled file is told: the game's tile
// properties and classes.
type tiledLoad struct {
	defs    []Property
	classes []Class
}

type tiledOption func(*tiledLoad)

func (o tiledOption) tiled(l *tiledLoad) { o(l) }
func (tiledOption) tileMap(*TileMap)     {}

// TileProperties declares a game's own tile properties: the loader checks
// their kinds and values, Value reads them with their defaults, and
// TiledProject writes them for Tiled.
func TileProperties(defs ...Property) TiledOption {
	return tiledOption(func(l *tiledLoad) { l.defs = append(l.defs, defs...) })
}

func loadOf(opts []TileMapOption) *tiledLoad {
	l := &tiledLoad{}
	for _, o := range opts {
		if t, ok := o.(TiledOption); ok {
			t.tiled(l)
		}
	}
	return l
}

// def is the definition of property name, the engine's or the game's.
func (l *tiledLoad) def(name string) (Property, bool) {
	for _, d := range EngineTileProperties() {
		if d.Name == name {
			return d, true
		}
	}
	for _, d := range l.defs {
		if d.Name == name {
			return d, true
		}
	}
	return Property{}, false
}

// check is value v, saved by Tiled as kind tiledType, read as d says, or
// what is wrong with it.
func (d Property) check(tiledType string, v any) (any, error) {
	if tiledType == "" {
		tiledType = typeString
	}
	switch d.Kind {
	case KindBool:
		b, ok := v.(bool)
		if !ok || tiledType != typeBool {
			return nil, fmt.Errorf("%v is a %s, not a bool", v, tiledType)
		}
		return b, nil
	case KindInt, KindFloat:
		f, ok := v.(float64)
		if !ok || tiledType != typeInt && tiledType != typeFloat || d.Kind == KindInt && f != float64(int(f)) {
			return nil, fmt.Errorf("%v is a %s, not %s %s", v, tiledType, article(kindNames[d.Kind]), kindNames[d.Kind])
		}
		if d.Min < d.Max && (f < d.Min || f > d.Max) {
			return nil, fmt.Errorf("%v is outside %v–%v", v, d.Min, d.Max)
		}
		if d.Kind == KindInt {
			return int(f), nil
		}
		return f, nil
	case KindColor:
		s, ok := v.(string)
		if !ok || tiledType != "color" && tiledType != typeString {
			return nil, fmt.Errorf("%v is a %s, not a color", v, tiledType)
		}
		return s, nil
	case KindString, KindEnum:
		s, ok := v.(string)
		if !ok || tiledType != typeString {
			return nil, fmt.Errorf("%v is a %s, not a string", v, tiledType)
		}
		if d.Kind == KindEnum && !slices.Contains(d.Values, s) {
			return nil, fmt.Errorf("%q is not one of %s", s, strings.Join(d.Values, ", "))
		}
		return s, nil
	case KindClass, KindFile, KindObject:
	}
	return nil, fmt.Errorf("no property kind %d", d.Kind)
}

func article(word string) string {
	if strings.ContainsRune("aeiou", rune(word[0])) {
		return "an"
	}
	return "a"
}

// Value is a tile property's value, as declared or as Tiled saved it.
type Value struct{ v any }

// Any is the value itself: a bool, an int, a float64 or a string.
func (v Value) Any() any { return v.v }

// String is the value as text.
func (v Value) String() string {
	if s, ok := v.v.(string); ok {
		return s
	}
	if v.v == nil {
		return ""
	}
	return fmt.Sprint(v.v)
}

// Int is the value as a whole number, 0 when it is none.
func (v Value) Int() int {
	switch n := v.v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

// Float is the value as a number, 0 when it is none.
func (v Value) Float() float64 {
	switch n := v.v.(type) {
	case int:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

// Bool is the value as true or false.
func (v Value) Bool() bool { b, _ := v.v.(bool); return b }

// Color is the value as a colour, Tiled's #aarrggbb or #rrggbb; white
// when it is none.
func (v Value) Color() color.Color {
	s, ok := v.v.(string)
	if !ok || len(s) == 0 {
		return color.White
	}
	if len(s) == 9 {
		s = "#" + s[3:] + s[1:3]
	}
	return color.Hex(s)
}

// Value is property name of tile id: as set in Tiled, else its class's
// default, else its declared default as a tile property; ok is false for a
// property neither set nor declared.
func (t *Tileset) Value(id int, name string) (Value, bool) {
	if v, ok := t.props[id][name]; ok {
		return Value{v}, true
	}
	if v, ok := t.load.tileValue(t.classes[id], name); ok {
		return Value{v}, true
	}
	if d, ok := t.declared(name); ok {
		return Value{d.Default}, true
	}
	return Value{}, false
}

func (t *Tileset) declared(name string) (Property, bool) {
	if t.load == nil {
		t.load = &tiledLoad{}
	}
	return t.load.def(name)
}

// Height is how tall tile id stands, in its pixels, for shadow length:
// its height property, else its picture's height when tall, its cell's
// when solid, else 0.
func (t *Tileset) Height(id int) float32 {
	if h, ok := t.height[id]; ok && h > 0 {
		return h
	}
	switch {
	case t.tall[id]:
		_, r := t.image(id, 0)
		return r.H
	case t.Solid(id):
		return float32(t.tileH)
	}
	return 0
}

// Emissive is how far tile id glows, in its pixels, 0 for not at all.
func (t *Tileset) Emissive(id int) float32 { return t.emissive[id] }

// NearMiss is the declared property name is within two edits of, when it
// isn't one itself: a likely misspelling.
func NearMiss(name string, defs ...Property) (string, bool) {
	all := append(EngineTileProperties(), defs...)
	for _, d := range all {
		if d.Name == name {
			return "", false
		}
	}
	for _, d := range all {
		if edits(name, d.Name) <= 2 {
			return d.Name, true
		}
	}
	return "", false
}

// edits is the Levenshtein distance from a to b.
func edits(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// TiledProject is a Tiled project file declaring what opts declare, by
// TileProperties and Classes: the engine's tile properties and the game's
// as an ember.Tile class, every class with its uses and its members at
// their defaults, a tile class with ember.Tile's members first, and an enum
// per enum property. Opened as the project in Tiled, maps, layers,
// objects and tiles show their properties as typed fields.
func TiledProject(opts ...TiledOption) []byte {
	load := &tiledLoad{}
	for _, o := range opts {
		o.tiled(load)
	}
	p := &tiledProject{}
	var tile []projectMember
	for _, d := range append(EngineTileProperties(), load.defs...) {
		tile = append(tile, p.member(d, d.Name))
	}
	p.add(projectType{Name: "ember.Tile", Type: typeClass, UseAs: []string{useTile}, Members: tile})
	for _, c := range load.classes {
		var members []projectMember
		if c.On&OnTile != 0 {
			members = append(members, tile...)
		}
		for _, d := range c.Properties {
			members = append(members, p.member(d, c.Name+"."+d.Name))
		}
		uses := []string{"property"}
		for _, u := range classUses {
			if c.On&u.use != 0 {
				uses = append(uses, u.name)
			}
		}
		p.add(projectType{Name: c.Name, Type: typeClass, UseAs: uses, Members: members})
	}
	b, _ := json.MarshalIndent(map[string]any{
		"automappingRulesFile": "",
		"commands":             []any{},
		"compatibilityVersion": 1100,
		"extensionsPath":       "extensions",
		"folders":              []string{"."},
		"propertyTypes":        p.types,
	}, "", "    ")
	return append(b, '\n')
}

// tiledProject are a project file's property types, as Tiled writes them.
type tiledProject struct{ types []projectType }

type projectMember struct {
	Name         string `json:"name"`
	PropertyType string `json:"propertyType,omitempty"`
	Type         string `json:"type"`
	Value        any    `json:"value"`
}

type projectType struct {
	Color         string          `json:"color,omitempty"`
	DrawFill      *bool           `json:"drawFill,omitempty"`
	ID            int             `json:"id"`
	Members       []projectMember `json:"members,omitempty"`
	Name          string          `json:"name"`
	StorageType   string          `json:"storageType,omitempty"`
	Type          string          `json:"type"`
	UseAs         []string        `json:"useAs,omitempty"`
	Values        []string        `json:"values,omitempty"`
	ValuesAsFlags *bool           `json:"valuesAsFlags,omitempty"`
}

// add numbers t and adds it, a class drawn as Tiled draws its own.
func (p *tiledProject) add(t projectType) {
	t.ID = len(p.types) + 1
	if t.Type == typeClass {
		yes := true
		t.Color, t.DrawFill = "#ffa0a0a4", &yes
	}
	p.types = append(p.types, t)
}

// member is d as a class member, an enum's type named enum and added.
func (p *tiledProject) member(d Property, enum string) projectMember {
	m := projectMember{Name: d.Name, Type: kindNames[d.Kind], Value: d.Default}
	switch d.Kind {
	case KindEnum:
		no := false
		p.add(projectType{Name: enum, StorageType: typeString, Type: "enum", Values: d.Values, ValuesAsFlags: &no})
		m.PropertyType = enum
	case KindClass:
		m.PropertyType, m.Value = d.Class, map[string]any{}
	case KindFloat:
		if f, ok := d.Default.(float64); ok && f == float64(int(f)) {
			m.Value = json.Number(fmt.Sprintf("%d", int(f)))
		}
	case KindFile:
		if d.Default == nil {
			m.Value = ""
		}
	case KindObject:
		if d.Default == nil {
			m.Value = 0
		}
	case KindBool, KindInt, KindString, KindColor:
	}
	return m
}
