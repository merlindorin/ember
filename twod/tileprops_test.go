package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"flag"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// propsTileset is a 2-tile atlas whose tile 0 carries props, as Tiled
// writes a property list.
func propsTileset(t *testing.T, props string) fstest.MapFS {
	return fstest.MapFS{
		"set.tsj": {
			Data: []byte(
				`{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "columns": 2, "tilecount": 2,
				  "tiles": [{"id": 0, "properties": [` + props + `]}]}`,
			),
		},
		"tiles.png": {Data: pngOf(t, 32, 16)},
	}
}

func TestPropertiesTwodDoesntDeclareKeepTheirTiledTypes(t *testing.T) {
	ts, err := LoadTiledTileset(propsTileset(t, `{"name": "solid", "type": "bool", "value": true},
		{"name": "shadow", "type": "string", "propertytype": "shadow", "value": "none"},
		{"name": "occluder_group", "type": "int", "value": 2},
		{"name": "height", "type": "float", "value": 24.5},
		{"name": "name", "type": "string", "value": "pillar"}`), "set.tsj")
	require.NoError(t, err)
	for name, want := range map[string]any{
		"solid": true, "shadow": "none", "occluder_group": 2, "height": 24.5, "name": "pillar",
	} {
		v, ok := ts.Value(1, name)
		assert.True(t, ok, name)
		assert.Equal(t, want, v.Any(), "%s, for the system that declares it to read", name)
	}
	_, ok := ts.Value(1, "missing")
	assert.False(t, ok, "neither set nor declared")
	_, ok = ts.Value(2, "solid")
	assert.False(t, ok, "solid is collide/tiles' to declare: undeclared here, no default")
}

func TestTheLoaderNamesTheTilesetTileAndPropertyThatAreWrong(t *testing.T) {
	const at = "set.tsj: tile 0: "
	for props, want := range map[string]string{
		`{"name": "tall", "type": "string", "value": "yes"}`:        at + "property tall: yes is a string, not a bool",
		`{"name": "reach_down", "type": "string", "value": "four"}`: "property reach_down: four is a string, not an int",
	} {
		_, err := LoadTiledTileset(
			propsTileset(t, props),
			"set.tsj",
			TileProperties(Property{Name: "reach_down", Kind: KindInt}),
		)
		require.Error(t, err, props)
		assert.Contains(t, err.Error(), want)
	}
}

func TestAGamesPropertiesReadTypedWithTheirDefaults(t *testing.T) {
	defs := TileProperties(
		Property{Name: "reach_down", Kind: KindInt, Default: 0},
		Property{Name: "tint", Kind: KindColor, Default: "#ff8000"},
	)
	ts, err := LoadTiledTileset(
		propsTileset(
			t,
			`{"name": "reach_down", "type": "int", "value": 4}, {"name": "tint", "type": "color", "value": "#ff00ff00"}`,
		),
		"set.tsj",
		defs,
	)
	require.NoError(t, err)
	v, _ := ts.Value(1, "reach_down")
	assert.Equal(t, 4, v.Int())
	c, _ := ts.Value(1, "tint")
	assert.Equal(t, color.Hex("#00ff00"), c.Color(), "Tiled's #aarrggbb read")
	d, ok := ts.Value(2, "reach_down")
	assert.True(t, ok, "a tile without it reads the default")
	assert.Equal(t, 0, d.Int())
}

func TestAMisspelledPropertyIsANearMiss(t *testing.T) {
	got, ok := NearMiss("tal")
	assert.True(t, ok)
	assert.Equal(t, "tall", got)
	_, ok = NearMiss("tall")
	assert.False(t, ok, "a known name is no miss")
	_, ok = NearMiss("reach_down")
	assert.False(t, ok, "far from every engine name")
	got, _ = NearMiss("reach_dwn", Property{Name: "reach_down", Kind: KindInt})
	assert.Equal(t, "reach_down", got, "a game's own too")
}

//nolint:gochecknoglobals // the -update flag go test passes
var updateProject = flag.Bool("update", false, "write the checked-in Tiled project files")
