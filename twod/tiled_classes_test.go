package twod //nolint:testpackage // white-box: shares twod's test helpers

import (
	"encoding/json"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crypt declares a game's classes: a level map, a zone layer, an exit
// object with a look and a sign, the look a class only properties take,
// and a wall tile.
func crypt() TiledOption {
	return Classes(
		Class{Name: "level", On: OnMap, Properties: []Property{
			{Name: "title", Kind: KindString, Default: "crypt"},
			{Name: "depth", Kind: KindInt, Default: 1, Min: 1, Max: 9},
		}},
		Class{Name: "zone", On: OnLayer, Properties: []Property{{Name: "lit", Kind: KindBool, Default: true}}},
		Class{Name: "exit", On: OnObject, Shapes: []TiledShape{TiledRectangle}, Properties: []Property{
			{Name: "to", Kind: KindString, Default: "", Required: true},
			{Name: "look", Kind: KindClass, Class: "look"},
			{Name: "sign", Kind: KindObject},
		}},
		Class{Name: "look", Properties: []Property{
			{Name: "glow", Kind: KindFloat, Default: 0.5, Min: 0, Max: 4},
			{Name: "tint", Kind: KindColor, Default: "#ffffffff"},
			{Name: "state", Kind: KindEnum, Default: "shut", Values: []string{"open", "shut"}},
		}},
		Class{Name: "wall", On: OnTile, Properties: []Property{
			{Name: "durability", Kind: KindInt, Default: 100, Min: 1, Max: 1000},
			{Name: "art", Kind: KindFile, Default: ""},
		}},
	)
}

func TestTheClassesProjectFileIsWhatTheDeclarationsWrite(t *testing.T) {
	const file = "tiled/classes/classes.tiled-project"
	if *updateProject {
		require.NoError(t, os.WriteFile(file, TiledProject(crypt()), 0o644))
	}
	b, err := os.ReadFile(file)
	require.NoError(t, err, "go test ./twod -run ProjectFile -update writes it")
	assert.Equal(t, string(TiledProject(crypt())), string(b), "regenerate with: go test ./twod -run ProjectFile -update")

	var project struct {
		PropertyTypes []struct {
			Name    string   `json:"name"`
			UseAs   []string `json:"useAs"`
			Members []struct {
				Name  string `json:"name"`
				Value any    `json:"value"`
			} `json:"members"`
		} `json:"propertyTypes"`
	}
	require.NoError(t, json.Unmarshal(b, &project))
	types := map[string][]string{}
	members := map[string]map[string]any{}
	for _, p := range project.PropertyTypes {
		types[p.Name] = p.UseAs
		members[p.Name] = map[string]any{}
		for _, m := range p.Members {
			members[p.Name][m.Name] = m.Value
		}
	}
	assert.Equal(t, []string{"property", "object"}, types["exit"])
	assert.Equal(t, []string{"property"}, types["look"], "a class only properties take")
	assert.Equal(t, []string{"property", "tile"}, types["wall"])
	assert.InDelta(t, 100, members["wall"]["durability"], 0, "each member at its default")
	assert.Contains(t, members["wall"], "solid", "a tile class shows every tile's properties too")
	assert.Contains(t, types, "look.state", "an enum member's values")
}

// TestAMapTiledSavedLoadsWithItsClassesFilled loads maps Tiled itself
// exported with the project above (tiled --project classes.tiled-project
// --export-map json): once as Tiled saves them, defaults left out, once
// with --resolve-types-and-properties, every default written. Tiled also
// keeps a member set to a value equal to its default; it reads the same.
func TestAMapTiledSavedLoadsWithItsClassesFilled(t *testing.T) {
	for _, name := range []string{"tiled/classes/crypt.tmj", "tiled/classes/crypt-resolved.tmj"} {
		m, err := LoadTiled(os.DirFS("."), name, crypt())
		require.NoError(t, err, name)

		title, _ := m.Properties().String("title")
		depth, _ := m.Properties().Int("depth")
		assert.Equal(t, "crypt", title, "%s: the map's class default", name)
		assert.Equal(t, 3, depth, "%s: and its own value", name)

		floor, _ := m.Layer("floor")
		lit, _ := floor.Properties.Bool("lit")
		assert.True(t, lit, "%s: a layer's class default", name)
		doors, _ := m.Layer("doors")
		lit, _ = doors.Properties.Bool("lit")
		assert.False(t, lit, "%s: a layer's own value", name)

		door, _ := m.Object(1)
		to, _ := door.Properties.String("to")
		assert.Equal(t, "crypt-2", to, name)
		look, ok := door.Properties.Class("look")
		require.True(t, ok, name)
		glow, _ := look.Float("glow")
		state, _ := look.String("state")
		assert.InDelta(t, 0.9, glow, 1e-9, "%s: a class-typed property's own member", name)
		assert.Equal(t, "shut", state, "%s: and its other members at their defaults", name)
		back, _ := m.Object(2)
		sign, _ := back.Properties.Int("sign")
		assert.Equal(t, 1, sign, "%s: an object property, the object's id", name)
		look, _ = back.Properties.Class("look")
		glow, _ = look.Float("glow")
		assert.InDelta(t, 0.5, glow, 1e-9, "%s: a class-typed property left out, at its class's defaults", name)

		walls := m.Tilesets()[0]
		assert.Equal(t, "wall", walls.Class(1), name)
		durability, _ := walls.Value(1, "durability")
		assert.Equal(t, 50, durability.Int(), "%s: a tile's own value", name)
		durability, _ = walls.Value(2, "durability")
		assert.Equal(t, 100, durability.Int(), "%s: a tile's class default, read when asked", name)
		solid, _ := walls.Value(2, "solid")
		assert.False(t, solid.Bool(), "%s: and every tile's properties beside it", name)
		art, _ := walls.Value(1, "art")
		assert.Equal(t, "tiled/classes/art/wall.png", art.String(), "%s: a file from the root", name)
	}
}

// classMap is a map of class level holding props, an object layer whose
// object is set by the test, and a tileset whose first tile is set too.
func classMap(object, tile string) fstest.MapFS {
	tmj := `{"orientation": "orthogonal", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "class": "level", "properties": [{"name": "depth", "type": "int", "value": 2}],
	 "tilesets": [{"firstgid": 1, "name": "t", "tilewidth": 16, "tileheight": 16,
	   "tiles": [{"id": 0` + tile + `}]}],
	 "layers": [{"type": "objectgroup", "name": "things", "objects": [{"id": 1, "x": 0, "y": 0, "width": 8,
	   "height": 8` + object + `}]}]}`
	return fstest.MapFS{"one.tmj": {Data: []byte(tmj)}}
}

func TestWhatADeclaredClassRefusesIsAnErrorNamingIt(t *testing.T) {
	exit := `, "type": "exit", "properties": [{"name": "to", "type": "string", "value": "hall"}`
	for _, c := range []struct{ name, object, tile, want string }{
		{"an undeclared class", `, "type": "exi"`, "", "object 1 (): class exi is not declared; is it exit?"},
		{"a class not for objects", `, "type": "wall"`, "", "class wall is not the class of a object"},
		{"a property its class lacks", exit + `, {"name": "too", "type": "string", "value": "x"}]`, "",
			"property too: class exit has no such property; is it to?"},
		{"a required property left out", `, "type": "exit"`, "", "property to: class exit requires it"},
		{"a required property left empty", `, "type": "exit", "properties": [{"name": "to", "type": "string",
		 "value": ""}]`, "", "property to: class exit requires it"},
		{"a wrong kind", `, "type": "exit", "properties": [{"name": "to", "type": "int", "value": 3}]`, "",
			"property to: 3 is a int, not a string"},
		{"a wrong shape", exit + `], "ellipse": true`, "", "class exit is a rectangle, not a ellipse"},
		{"a member out of range", exit + `, {"name": "look", "type": "class", "propertytype": "look",
		 "value": {"glow": 9}}]`, "", "property look: member glow: 9 is outside 0–4"},
		{"a member its class lacks", exit + `, {"name": "look", "type": "class", "propertytype": "look",
		 "value": {"glo": 1}}]`, "", "member glo: class look has no such property; is it glow?"},
		{"a class property of another class", exit + `, {"name": "look", "type": "class", "propertytype": "level",
		 "value": {}}]`, "", "property look: map[] is a level, not a look"},
		{"an enum member's value", exit + `, {"name": "look", "type": "class", "propertytype": "look",
		 "value": {"state": "ajar"}}]`, "", `member state: "ajar" is not one of open, shut`},
		{"a tile's undeclared class", "", `, "type": "wal"`, "tile 0: class wal is not declared; is it wall?"},
		{"a tile property neither its class nor every tile has", "", `, "type": "wall",
		 "properties": [{"name": "durabilty", "type": "int", "value": 3}]`,
			"tile 0: property durabilty: class wall has no such property; is it durability?"},
		{"a tile value out of range", "", `, "type": "wall",
		 "properties": [{"name": "durability", "type": "int", "value": 0}]`,
			"tile 0: property durability: 0 is outside 1–1000"},
	} {
		_, err := LoadTiled(classMap(c.object, c.tile), "one.tmj", crypt())
		require.Error(t, err, c.name)
		assert.Contains(t, err.Error(), c.want, c.name)
	}
}

func TestElementsOfNoDeclaredClassStayAsTiledSavedThem(t *testing.T) {
	m, err := LoadTiled(classMap(`, "properties": [{"name": "loot", "type": "string", "value": "coins"},
	 {"name": "look", "type": "class", "propertytype": "look", "value": {"glow": 2}}]`,
		`, "properties": [{"name": "loot", "type": "int", "value": 4}]`), "one.tmj", crypt())
	require.NoError(t, err)
	o, _ := m.Object(1)
	loot, _ := o.Properties.String("loot")
	assert.Equal(t, "coins", loot, "an object of no class keeps what it has")
	look, _ := o.Properties.Class("look")
	state, _ := look.String("state")
	assert.Equal(t, "shut", state, "a property of a declared class is still filled")
	n, _ := m.Tilesets()[0].Value(1, "loot")
	assert.Equal(t, 4, n.Int(), "a tile of no class keeps its own properties")

	loose, err := LoadTiled(classMap(`, "type": "anything", "properties": [{"name": "x", "type": "int",
	 "value": 1}]`, `, "type": "whatever"`), "one.tmj")
	require.NoError(t, err, "nothing declared: classes are names only")
	o, _ = loose.Object(1)
	assert.Equal(t, "anything", o.Type)

	onlyMaps, err := LoadTiled(classMap(`, "type": "anything"`, ""), "one.tmj",
		Classes(Class{Name: "level", On: OnMap, Properties: []Property{{Name: "depth", Kind: KindInt}}}))
	require.NoError(t, err, "no class declared for objects: an object's class is a name only")
	o, _ = onlyMaps.Object(1)
	assert.Equal(t, "anything", o.Type)
}

func TestAClassedTilesMembersAreCheckedLikeEveryTilesProperties(t *testing.T) {
	m, err := LoadTiled(classMap("", `, "type": "wall", "properties": [{"name": "solid", "type": "bool",
	 "value": true}, {"name": "durability", "type": "int", "value": 7}]`), "one.tmj", crypt())
	require.NoError(t, err)
	walls := m.Tilesets()[0]
	assert.True(t, walls.Solid(1), "an engine property on a classed tile still applies")
	d, _ := walls.Value(1, "durability")
	assert.Equal(t, 7, d.Int())
	_, err = LoadTiled(classMap("", `, "type": "wall", "properties": [{"name": "solid", "type": "int",
	 "value": 1}]`), "one.tmj", crypt())
	require.Error(t, err, "an engine property of the wrong kind is refused on a classed tile too")
	assert.Contains(t, err.Error(), "tile 0: property solid: 1 is a int, not a bool")
}
