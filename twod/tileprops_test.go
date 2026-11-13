package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"flag"
	"os"
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

func TestTheLoaderReadsTheEngineTilePropertiesTyped(t *testing.T) {
	ts, err := LoadTiledTileset(propsTileset(t, `{"name": "solid", "type": "bool", "value": true},
		{"name": "shadow", "type": "string", "propertytype": "shadow", "value": "none"},
		{"name": "occluder_group", "type": "int", "value": 2},
		{"name": "height", "type": "float", "value": 24.5},
		{"name": "emissive", "type": "int", "value": 6},
		{"name": "name", "type": "string", "value": "pillar"}`), "set.tsj")
	require.NoError(t, err)
	assert.True(t, ts.Solid(1))
	assert.False(t, ts.CastShadow(1))
	assert.Equal(t, 2, ts.OccluderGroup(1))
	assert.Equal(t, float32(24.5), ts.Height(1))
	assert.Equal(t, float32(6), ts.Emissive(1))
	v, ok := ts.Value(1, "name")
	assert.True(t, ok)
	assert.Equal(t, "pillar", v.String())
	_, ok = ts.Value(1, "missing")
	assert.False(t, ok, "neither set nor declared")
	assert.Equal(t, float32(0), ts.Height(2), "a floor stands flat")
}

func TestTheLoaderNamesTheTilesetTileAndPropertyThatAreWrong(t *testing.T) {
	const at = "set.tsj: tile 0: "
	for props, want := range map[string]string{
		`{"name": "tall", "type": "string", "value": "yes"}`:        at + "property tall: yes is a string, not a bool",
		`{"name": "shadow", "type": "string", "value": "half"}`:     at + `property shadow: "half" is not one of cast, none`,
		`{"name": "occluder_group", "type": "int", "value": 3}`:     at + "property occluder_group: 3 is outside 0–2",
		`{"name": "occluder_group", "type": "float", "value": 1.5}`: "property occluder_group: 1.5 is a float, not an int",
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
	got, ok := NearMiss("occluder_grup")
	assert.True(t, ok)
	assert.Equal(t, "occluder_group", got)
	_, ok = NearMiss("shadow")
	assert.False(t, ok, "a known name is no miss")
	_, ok = NearMiss("reach_down")
	assert.False(t, ok, "far from every engine name")
	got, _ = NearMiss("reach_dwn", Property{Name: "reach_down", Kind: KindInt})
	assert.Equal(t, "reach_down", got, "a game's own too")
}

//nolint:gochecknoglobals // the -update flag go test passes
var updateProject = flag.Bool("update", false, "write tiled/ember.tiled-project")

func TestTheCheckedInTiledProjectIsTheEnginesProperties(t *testing.T) {
	if *updateProject {
		require.NoError(t, os.WriteFile("tiled/ember.tiled-project", TiledProject(), 0o644))
	}
	b, err := os.ReadFile("tiled/ember.tiled-project")
	require.NoError(t, err, "go test ./twod -run TiledProject -update writes it")
	assert.Equal(t, string(TiledProject()), string(b), "regenerate with: go test ./twod -run TiledProject -update")
}
