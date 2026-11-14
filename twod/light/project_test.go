package light //nolint:testpackage // beside light's other tests

import (
	"bytes"
	"flag"
	"image"
	imagepng "image/png"
	"os"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/collide/tiles"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals // the -update flag go test passes
var updateProject = flag.Bool("update", false, "write twod/tiled/ember.tiled-project")

// The engine's Tiled project declares every tile property the engine
// reads: twod's tall, collide/tiles' solid, light's shadow, occluder_group
// and height. It lives beside twod's tilesets; light writes it, as it can
// import every system that declares one.
func TestTheCheckedInTiledProjectIsTheEnginesProperties(t *testing.T) {
	const file = "../tiled/ember.tiled-project"
	want := twod.TiledProject(tiles.Properties(), TileProperties())
	if *updateProject {
		require.NoError(t, os.WriteFile(file, want, 0o644))
	}
	b, err := os.ReadFile(file)
	require.NoError(t, err, "go test ./twod/light -run TiledProject -update writes it")
	assert.Equal(t, string(want), string(b), "regenerate with: go test ./twod/light -run TiledProject -update")
}

// files are a tileset, set.tsj, of tsj, and its 16×16 picture.
func files(t *testing.T, tsj string) fstest.MapFS {
	t.Helper()
	var png bytes.Buffer
	require.NoError(t, imagepng.Encode(&png, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	return fstest.MapFS{"set.tsj": {Data: []byte(tsj)}, "tiles.png": {Data: png.Bytes()}}
}

func TestLightRefusesItsPropertiesSetWrongWhenDeclared(t *testing.T) {
	for props, want := range map[string]string{
		`{"name": "shadow", "type": "string", "value": "half"}`:     `property shadow: "half" is not one of cast, none`,
		`{"name": "occluder_group", "type": "int", "value": 3}`:     "property occluder_group: 3 is outside 0–2",
		`{"name": "occluder_group", "type": "float", "value": 1.5}`: "property occluder_group: 1.5 is a float, not an int",
		`{"name": "height", "type": "string", "value": "tall"}`:     "property height: tall is a string, not a float",
	} {
		tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "columns": 1, "tilecount": 1,
		 "tiles": [{"id": 0, "properties": [` + props + `]}]}`
		_, err := twod.LoadTiledTileset(files(t, tsj), "set.tsj", TileProperties())
		require.Error(t, err, props)
		assert.Contains(t, err.Error(), "set.tsj: tile 0: "+want)
	}
}
