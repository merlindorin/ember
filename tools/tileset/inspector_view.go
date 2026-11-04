package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/twod"
)

// The inspector is a view the Tool drives: it shows an inspectorState and
// reports what the player asks for through inspectorIntents, never
// touching the editor itself. The ui view in inspector_doc.go
// implements it; the tests drive it through automation alone.
//
// Automation IDs, the leaves of the inspector's paths: the properties
// solid and tall (toggles, each with a <name>-default label while unset),
// shadow and occluder_group (combos), height and emissive (text inputs),
// animation and near-misses (labels), all under a node named properties;
// under one named objects, each folded object's object-<n>, pressed to
// select it, and its object-<n>-summary (a label of its line), n from 1
// in file order, then the open object's title (label), type and shape
// (combos), x, y, width, height and points (text inputs); add-object (a
// combo, placeholder "+ Add object") and delete (a button). The header
// is tile/title, and the whole a node named inspector, whose bounds
// nothing inside passes.

// inspectorState is everything the inspector shows of the picked tile,
// or of the tiles marked with it.
type inspectorState struct {
	// Title is "Tile <id>", "Tile <id> · <class>" when the tile has a
	// class, or "<n> tiles" with several marked.
	Title string
	// Loaded is false while no tileset is open.
	Loaded     bool
	Properties []propertyState
	// Animation is the tile's frames as text, "none" without; NearMisses
	// name its properties misspelling an engine one.
	Animation  string
	NearMisses []string
	Objects    []objectState
	// Selected is the open object, -1 for none.
	Selected int
}

// propertyState is an engine property of the tile: its Value as text
// ("true", "cast", "2", "64", "" for an empty field), whether the file
// sets it, whether the marked tiles differ (Mixed, shown "—"), and its
// default as text.
type propertyState struct {
	Name    string
	Kind    twod.PropertyKind
	Value   string
	Set     bool
	Mixed   bool
	Default string
	// Values are a pop-up's items: an enum's values, the occluder
	// group's 0, 1, 2; nil for a switch or a field.
	Values []string
	// Hint is a field's placeholder while empty, such as the height the
	// engine gives the tile.
	Hint string
}

// objectState is an object of the tile's objectgroup, in file order: its
// id, Type (none, shadow, body), Shape (rectangle, ellipse, polygon), its
// place and size, and a polygon's points from X, Y.
type objectState struct {
	ID         int
	Type       string
	Shape      string
	X, Y, W, H float32
	Points     []geom.Vec2
}

// inspectorIntents is what an inspector view asks the Tool for.
type inspectorIntents interface {
	SetBool(name string, on bool)
	// SetChoice picks item index of property name's pop-up.
	SetChoice(name string, index int)
	// SetField sets property name from text typed in its field.
	SetField(name, text string)
	SelectObject(i int)
	// ObjectChoice picks item index of object i's "type" or "shape".
	ObjectChoice(i int, key string, index int)
	// ObjectField sets object i's "x", "y", "width", "height" or
	// "points" from text typed in its field.
	ObjectField(i int, key, text string)
	AddObject(kind Kind)
	DeleteObject()
}

// objectTypes and objectKinds are an object's type and shape, as the
// file names them.
//
//nolint:gochecknoglobals // word tables
var (
	objectTypes = []string{noClass, classShadow, classBody}
	objectKinds = []string{"rectangle", "ellipse", keyPolygon}
)

// objectField is which field of which object a control edits.
type objectField struct {
	i   int
	key string
}

// engineProperties are the tile properties the engine reads, as the
// inspector shows them.
func engineProperties() []twod.TileProperty { return twod.EngineTileProperties() }

// valuesOf are the items of property d's pop-up: the occluder group's 0,
// 1, 2, an enum's values, nil for a bool's switch or a field.
func valuesOf(d twod.TileProperty) []string {
	switch {
	case d.Name == propOccluderGroup:
		return []string{"0", "1", "2"}
	case d.Kind == twod.KindEnum:
		return d.Values
	}
	return nil
}

// newUIInspector builds the inspector in a panel section makes, telling
// intents what the player asks for.

// summary is object o's line while folded: its type, shape, place and
// size, or a polygon's points.
func summary(o objectState) string {
	f := func(x float32) string { return strconv.FormatFloat(float64(x), 'g', -1, 32) }
	where := fmt.Sprintf("%s,%s · %s×%s", f(o.X), f(o.Y), f(o.W), f(o.H))
	if o.Shape == keyPolygon {
		where = fmt.Sprintf("%s,%s · %d points", f(o.X), f(o.Y), len(o.Points))
	}
	return strings.Join([]string{o.Type, o.Shape, where}, " · ")
}
