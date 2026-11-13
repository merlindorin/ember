package twod //nolint:testpackage // white-box: shares twod's test helpers

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tiledKey is what the loader does with one of a Tiled file's keys.
type tiledKey int

const (
	keyApplied tiledKey = iota
	keyExposed
	keyRefused
	keyNotRead
)

// tiledKeys are the keys each part of a Tiled file may hold, by what the
// loader does with them: applied, exposed, refused, or not read because
// only the editor uses them. A key in none of these is one the loader
// would drop without a word.
//
//nolint:gochecknoglobals // the test's lists
var tiledKeys = map[string]map[string]tiledKey{
	"map": {
		"orientation": keyApplied, "tilewidth": keyApplied, "tileheight": keyApplied, "layers": keyApplied,
		"tilesets": keyApplied,
		"class":    keyExposed, "width": keyExposed, "height": keyExposed, "backgroundcolor": keyExposed,
		"parallaxoriginx": keyExposed, "parallaxoriginy": keyExposed, "properties": keyExposed,
		"infinite": keyRefused, "staggeraxis": keyRefused, "staggerindex": keyRefused, "hexsidelength": keyRefused,
		"type": keyNotRead, "version": keyNotRead, "tiledversion": keyNotRead, "nextlayerid": keyNotRead,
		"nextobjectid": keyNotRead, "compressionlevel": keyNotRead, "editorsettings": keyNotRead,
		"renderorder": keyNotRead,
	},
	"layer": {
		"type": keyApplied, "data": keyApplied, "encoding": keyApplied, "compression": keyApplied,
		"width": keyApplied, "height": keyApplied, "objects": keyApplied,
		"id": keyExposed, "name": keyExposed, "class": keyExposed, "visible": keyExposed, "opacity": keyExposed,
		"offsetx": keyExposed, "offsety": keyExposed, "tintcolor": keyExposed, "parallaxx": keyExposed,
		"parallaxy": keyExposed, "draworder": keyExposed, "properties": keyExposed,
		"layers": keyRefused, "image": keyRefused, "imagewidth": keyRefused, "imageheight": keyRefused,
		"repeatx": keyRefused, "repeaty": keyRefused, "transparentcolor": keyRefused, "chunks": keyRefused,
		"startx": keyRefused, "starty": keyRefused,
		"x": keyNotRead, "y": keyNotRead, "locked": keyNotRead,
	},
	"object": {
		"gid": keyApplied, "ellipse": keyApplied, "point": keyApplied, "polygon": keyApplied, "polyline": keyApplied,
		"id": keyExposed, "name": keyExposed, "type": keyExposed, "class": keyExposed, "x": keyExposed,
		"y": keyExposed, "width": keyExposed, "height": keyExposed, "rotation": keyExposed, "visible": keyExposed,
		"properties": keyExposed,
		"template":   keyApplied, "text": keyExposed,
	},
	"tileset": {
		"firstgid": keyApplied, "source": keyApplied, "image": keyApplied, "tilewidth": keyApplied,
		"tileheight": keyApplied, "spacing": keyApplied, "margin": keyApplied, "tiles": keyApplied,
		"grid": keyApplied, "objectalignment": keyApplied,
		"tileoffset": keyApplied, "tilerendersize": keyRefused, "fillmode": keyRefused,
		"transparentcolor": keyRefused, "class": keyRefused, "properties": keyRefused,
		"name": keyNotRead, "type": keyNotRead, "version": keyNotRead, "tiledversion": keyNotRead,
		"columns": keyNotRead, "tilecount": keyNotRead, "imagewidth": keyNotRead, "imageheight": keyNotRead,
		"editorsettings": keyNotRead, "transformations": keyNotRead, "wangsets": keyNotRead, "terrains": keyNotRead,
	},
	"tile": {
		"id": keyApplied, "image": keyApplied, "x": keyApplied, "y": keyApplied, "width": keyApplied,
		"height": keyApplied, "properties": keyApplied, "animation": keyApplied, "objectgroup": keyApplied,
		"type": keyRefused, "class": keyRefused,
		"imagewidth": keyNotRead, "imageheight": keyNotRead, "probability": keyNotRead, "terrain": keyNotRead,
	},
	"collision": {
		"objects":   keyApplied,
		"draworder": keyNotRead, "id": keyNotRead, "name": keyNotRead, "opacity": keyNotRead, "type": keyNotRead,
		"visible": keyNotRead, "x": keyNotRead, "y": keyNotRead,
	},
	"shape": {
		"x": keyApplied, "y": keyApplied, "width": keyApplied, "height": keyApplied, "polygon": keyApplied,
		"ellipse": keyApplied, "point": keyApplied, "class": keyApplied, "type": keyApplied,
		"rotation": keyRefused,
		"id":       keyNotRead, "name": keyNotRead, "visible": keyNotRead,
	},
	"text": {
		"text": keyExposed, "fontfamily": keyExposed, "pixelsize": keyExposed, "color": keyExposed, "wrap": keyExposed,
		"bold": keyExposed, "italic": keyExposed, "underline": keyExposed, "strikeout": keyExposed,
		"kerning": keyExposed, "halign": keyExposed, "valign": keyExposed,
	},
	"template": {"object": keyApplied, "tileset": keyApplied, "type": keyNotRead},
	"property": {"name": keyApplied, "type": keyApplied, "value": keyApplied, "propertytype": keyApplied},
	"frame":    {"tileid": keyApplied, "duration": keyApplied},
	"grid":     {"width": keyApplied, "height": keyApplied, "orientation": keyNotRead},
	"point":    {"x": keyApplied, "y": keyApplied},
}

// tiledParts are the parts a part of a Tiled file holds, by key.
//
//nolint:gochecknoglobals // the test's lists
var tiledParts = map[string]map[string]string{
	"map":       {"layers": "layer", "tilesets": "tileset", "properties": "property"},
	"layer":     {"objects": "object", "properties": "property", "layers": "layer"},
	"object":    {"properties": "property", "polygon": "point", "polyline": "point", "text": "text"},
	"template":  {"object": "object", "tileset": "tileset"},
	"tileset":   {"tiles": "tile", "properties": "property", "grid": "grid", "tileoffset": "point"},
	"tile":      {"properties": "property", "animation": "frame", "objectgroup": "collision"},
	"collision": {"objects": "shape"},
	"shape":     {"polygon": "point"},
}

// unlistedKeys are the keys of v, a part of a Tiled file of kind, and of
// the parts it holds, that no list names, each with its path.
func unlistedKeys(kind, at string, v any) []string {
	switch v := v.(type) {
	case []any:
		var out []string
		for _, e := range v {
			out = append(out, unlistedKeys(kind, at, e)...)
		}
		return out
	case map[string]any:
		var out []string
		for key, value := range v {
			if _, ok := tiledKeys[kind][key]; !ok {
				out = append(out, at+"."+key)
				continue
			}
			if part, ok := tiledParts[kind][key]; ok {
				out = append(out, unlistedKeys(part, at+"."+key, value)...)
			}
		}
		return out
	}
	return nil
}

func TestEveryKeyOfEmbersTiledFilesIsAppliedExposedRefusedOrListedNotRead(t *testing.T) {
	var files []string
	require.NoError(t, filepath.WalkDir("..", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(p, ".tmj") || strings.HasSuffix(p, ".tsj") || strings.HasSuffix(p, ".tj")) {
			files = append(files, p)
		}
		return nil
	}))
	require.NotEmpty(t, files, "ember's own maps and tilesets")
	for _, f := range files {
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		var v any
		require.NoError(t, json.Unmarshal(data, &v), f)
		kind := "map"
		switch {
		case strings.HasSuffix(f, ".tsj"):
			kind = "tileset"
		case strings.HasSuffix(f, ".tj"):
			kind = "template"
		}
		assert.Empty(t, unlistedKeys(kind, f, v), "%s holds keys the loader would drop silently", f)
	}
}

func TestAKeyNoListNamesIsCaught(t *testing.T) {
	var v any
	require.NoError(t, json.Unmarshal([]byte(`{"orientation": "orthogonal", "shiny": true,
	 "layers": [{"type": "objectgroup", "objects": [{"id": 1, "glow": 2}]}],
	 "tilesets": [{"firstgid": 1, "tiles": [{"id": 0, "animation": [{"tileid": 0, "speed": 3}]}]}]}`), &v))
	got := unlistedKeys("map", "m", v)
	slices.Sort(got)
	assert.Equal(t, []string{"m.layers.objects.glow", "m.shiny", "m.tilesets.tiles.animation.speed"}, got)
}

func TestAMapUsingEveryKeyTheLoaderTakesLoads(t *testing.T) {
	tmj := `{"type": "map", "version": "1.10", "tiledversion": "1.11.2", "orientation": "orthogonal",
	 "renderorder": "right-down", "compressionlevel": -1, "nextlayerid": 4, "nextobjectid": 3, "infinite": false,
	 "class": "level", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16, "backgroundcolor": "#202020",
	 "parallaxoriginx": 0, "parallaxoriginy": 0, "editorsettings": {"export": {"format": "json"}},
	 "properties": [{"name": "tier", "type": "string", "propertytype": "tier", "value": "crypt"}],
	 "tilesets": [{"firstgid": 1, "name": "tiles", "type": "tileset", "version": "1.10", "tiledversion": "1.11.2",
	   "image": "tiles.png", "imagewidth": 32, "imageheight": 16, "tilewidth": 16, "tileheight": 16,
	   "columns": 2, "tilecount": 2, "spacing": 0, "margin": 0, "objectalignment": "bottomleft",
	   "grid": {"orientation": "orthogonal", "width": 16, "height": 16}, "transformations": {"hflip": true},
	   "wangsets": [],
	   "tiles": [{"id": 0, "probability": 0.5, "properties": [{"name": "solid", "type": "bool", "value": true}],
	     "animation": [{"tileid": 0, "duration": 100}, {"tileid": 1, "duration": 100}],
	     "objectgroup": {"type": "objectgroup", "name": "", "id": 2, "draworder": "index", "opacity": 1,
	       "visible": true, "x": 0, "y": 0, "objects": [{"id": 1, "name": "", "visible": true, "class": "shadow",
	       "type": "", "x": 0, "y": 8, "width": 16, "height": 8}]}}]}],
	 "layers": [
	  {"id": 1, "type": "tilelayer", "name": "floor", "class": "", "x": 0, "y": 0, "width": 1, "height": 1,
	   "visible": true, "locked": false, "opacity": 1, "offsetx": 0, "offsety": 0, "tintcolor": "#ffffff",
	   "parallaxx": 1, "parallaxy": 1, "data": [1]},
	  {"id": 2, "type": "objectgroup", "name": "things", "draworder": "topdown", "x": 0, "y": 0, "visible": true,
	   "opacity": 1, "objects": [
	    {"id": 1, "name": "spot", "type": "", "class": "spawn", "x": 4, "y": 4, "width": 0, "height": 0,
	     "rotation": 0, "visible": true, "point": true},
	    {"id": 2, "name": "pool", "class": "", "x": 0, "y": 0, "width": 8, "height": 8, "rotation": 30,
	     "visible": true, "ellipse": true,
	     "properties": [{"name": "deep", "type": "bool", "value": true}]}]}
	 ]}`
	var v any
	require.NoError(t, json.Unmarshal([]byte(tmj), &v))
	require.Empty(t, unlistedKeys("map", "m", v), "the test's map holds only listed keys")
	_, err := LoadTiled(fstest.MapFS{"one.tmj": {Data: []byte(tmj)}, "tiles.png": {Data: pngOf(t, 32, 16)}},
		"one.tmj")
	require.NoError(t, err)
}
