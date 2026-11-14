package twod //nolint:testpackage // white-box: reads the pictures cells draw

import (
	"bytes"
	"image"
	stdcolor "image/color"
	"image/png"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The colours of tiled/variants/tiles.png: tile 0 is water with a deep
// corner (its top-left 4×4 pixels), tile 1 stone. masks.png's tile half
// keeps the right half; hole is clear within 4 pixels of the middle, at
// half alpha out to 6, opaque beyond.
//
//nolint:gochecknoglobals // the fixture's colours
var (
	waterRGB = stdcolor.NRGBA{0x68, 0x6f, 0x99, 0xff}
	deepRGB  = stdcolor.NRGBA{0x40, 0x49, 0x73, 0xff}
	stoneRGB = stdcolor.NRGBA{0x80, 0x80, 0x80, 0xff}
	darkRGB  = stdcolor.NRGBA{0x2c, 0x35, 0x4d, 0xff}
	nightRGB = stdcolor.NRGBA{0x14, 0x18, 0x2e, 0xff}
	blackRGB = stdcolor.NRGBA{0x10, 0x10, 0x10, 0xff}
)

// variantPalettes are the palettes the variants fixture names: dark turns
// the water and its deep corner darker, deep only the water black.
func variantPalettes() map[string]*Palette {
	hex := func(c stdcolor.NRGBA) color.Color { return color.RGB8(c.R, c.G, c.B) }
	return map[string]*Palette{
		"dark": NewPalette(Swap(hex(waterRGB), hex(darkRGB)), Swap(hex(deepRGB), hex(nightRGB))),
		"deep": NewPalette(Swap(hex(waterRGB), hex(blackRGB))),
	}
}

// variantsFS is tiled/variants, with any of its files replaced by over.
func variantsFS(t testing.TB, over map[string]string) fs.FS {
	t.Helper()
	m := fstest.MapFS{}
	for _, f := range []string{"tiles.png", "masks.png", "tiles.tsj", "masks.tsj", "variants.tmj"} {
		b, err := os.ReadFile("tiled/variants/" + f)
		require.NoError(t, err)
		m[f] = &fstest.MapFile{Data: b}
	}
	for f, s := range over {
		m[f] = &fstest.MapFile{Data: []byte(s)}
	}
	return m
}

// loadVariants loads variants.tmj of fsys with the fixture's palettes and
// its mask tileset, and opts.
func loadVariants(t testing.TB, fsys fs.FS, opts ...TileMapOption) (*Tiled, error) {
	t.Helper()
	masks, err := LoadTiledTileset(fsys, "masks.tsj")
	require.NoError(t, err)
	return LoadTiled(fsys, "variants.tmj", append([]TileMapOption{Palettes(variantPalettes()), Masks(masks)}, opts...)...)
}

// pixel is pixel x, y of tex's region r.
func pixel(tex *Texture, x, y int, at ...float32) stdcolor.NRGBA {
	ox, oy := 0, 0
	if len(at) == 2 {
		ox, oy = int(at[0]), int(at[1])
	}
	c, _ := stdcolor.NRGBAModel.Convert(tex.Image.At(ox+x, oy+y)).(stdcolor.NRGBA)
	return c
}

// cellPixel is pixel x, y of what cell i of m draws.
func cellPixel(m *TileMap, i, x, y int) stdcolor.NRGBA {
	id := m.cells[i]
	tex, r := m.tileset.drawn(id, 0, m.cellLook(i, id))
	return pixel(tex, x, y, r.X, r.Y)
}

// cellTint is the tint cell i of m draws its tile with.
func cellTint(m *TileMap, i int) color.Color { return tintOf(m.cellLook(i, m.cells[i])) }

func tiled(t *testing.T, s string) color.Color {
	t.Helper()
	c, err := tiledColor(s)
	require.NoError(t, err)
	return c
}

// solidImage is a w×h picture all of c.
func solidImage(w, h int, c stdcolor.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func pngImage(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func alpha(c stdcolor.NRGBA, a uint8) stdcolor.NRGBA { c.A = a; return c }

func TestAMapMadeInTiledDrawsItsTintsRecoloursAndMasks(t *testing.T) {
	level, err := loadVariants(t, os.DirFS("tiled/variants"))
	require.NoError(t, err)
	floor, ok := level.TileLayer("floor")
	require.True(t, ok)

	assert.Equal(t, nightRGB, cellPixel(floor, 1, 2, 2), "the layer's recolour: the deep corner")
	assert.Equal(t, darkRGB, cellPixel(floor, 1, 12, 12), "the layer's recolour: the water")
	assert.Equal(t, alpha(nightRGB, 0), cellPixel(floor, 0, 2, 2), "recoloured, then cut where the mask is clear")
	assert.Equal(t, darkRGB, cellPixel(floor, 0, 12, 2), "kept where it is opaque")
	assert.Equal(t, nightRGB, cellPixel(floor, 4, 2, 2), "a mask cell flipped across keeps the other half")
	assert.Equal(t, alpha(darkRGB, 0), cellPixel(floor, 4, 12, 2))

	assert.Equal(t, stoneRGB, cellPixel(floor, 2, 0, 0), "stone holds none of the palette's colours")
	assert.Equal(t, alpha(stoneRGB, 0), cellPixel(floor, 2, 7, 7), "the hole")
	assert.Equal(t, alpha(stoneRGB, 0x80), cellPixel(floor, 2, 7, 3), "its rim at half alpha halves the tile's")
	assert.Equal(t, tiled(t, "#ffc08080"), cellTint(floor, 2), "the tile's own tint, kept under a cut")
	assert.Equal(t, tiled(t, "#ffc08080"), cellTint(floor, 3), "and where nothing cuts it")

	cut, ok := level.Layer("cut")
	require.True(t, ok)
	assert.Equal(t, TiledMask, cut.Kind)
	assert.Nil(t, cut.TileMap(), "a mask layer is never drawn")
	assert.Len(t, level.TileLayers(), 1)

	objects := level.Objects("props")
	pool := level.Sprite(objects[0])
	assert.Equal(t, blackRGB, pixel(pool.Texture(), 12, 12, pool.Region().X, pool.Region().Y), "the object's recolour")
	assert.Equal(t, deepRGB, pixel(pool.Texture(), 2, 2, pool.Region().X, pool.Region().Y), "deep swaps only water")
	pillar := level.Sprite(objects[1])
	assert.Equal(t, alpha(stoneRGB, 0x80), pixel(pillar.Texture(), 7, 3, pillar.Region().X, pillar.Region().Y),
		"the object's mask")
	assert.Equal(t, tiled(t, "#ffc08080").Mul(tiled(t, "#ff8080ff")), pillar.Tint(), "the tile's tint times the object's")
}

func TestATilesOwnRecolourMaskAndTintDrawWhereverItIs(t *testing.T) {
	tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "tiles": [{"id": 0, "properties": [
	 {"name": "recolour", "type": "string", "value": "dark"}, {"name": "mask", "type": "string", "value": "half"},
	 {"name": "tint", "type": "color", "value": "#80ff0000"}]}]}`
	tmj := `{"orientation": "orthogonal", "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tiles.tsj"}],
	 "layers": [{"id": 1, "type": "tilelayer", "name": "floor", "width": 2, "height": 1, "data": [1, 2]},
	  {"id": 2, "type": "objectgroup", "name": "props", "objects": [{"id": 1, "gid": 1, "x": 0, "y": 16,
	   "width": 16, "height": 16}]}]}`
	level, err := loadVariants(t, variantsFS(t, map[string]string{"tiles.tsj": tsj, "variants.tmj": tmj}))
	require.NoError(t, err)
	floor, _ := level.TileLayer("floor")
	assert.Equal(t, alpha(nightRGB, 0), cellPixel(floor, 0, 2, 2), "recoloured, then masked")
	assert.Equal(t, darkRGB, cellPixel(floor, 0, 12, 2))
	assert.Equal(t, tiled(t, "#80ff0000"), cellTint(floor, 0), "then tinted, when drawn")
	assert.Equal(t, stoneRGB, cellPixel(floor, 1, 2, 2), "another tile stays as it is")
	assert.Equal(t, color.White, cellTint(floor, 1))

	s := level.Sprite(level.Objects("props")[0])
	assert.Equal(t, darkRGB, pixel(s.Texture(), 12, 2, s.Region().X, s.Region().Y), "its tile object too")
	assert.Equal(t, tiled(t, "#80ff0000"), s.Tint())

	ts := floor.Tileset()
	other := NewTileMap(ts, Cells([][]int{{1}}))
	assert.Equal(t, darkRGB, cellPixel(other, 0, 12, 2), "and a map made in code of the tileset")
}

func TestALayersTintAndRecolourDrawItsTilesItsPictureAndItsObjects(t *testing.T) {
	tmj := `{"orientation": "orthogonal", "width": 2, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tiles.tsj"}],
	 "layers": [
	  {"id": 1, "type": "tilelayer", "name": "floor", "width": 2, "height": 1, "data": [1, 2],
	   "tintcolor": "#ff808080", "properties": [{"name": "tint", "type": "color", "value": "#ff00ff00"},
	   {"name": "recolour", "type": "string", "value": "dark"}]},
	  {"id": 2, "type": "imagelayer", "name": "sky", "image": "tiles.png",
	   "properties": [{"name": "recolour", "type": "string", "value": "deep"}]},
	  {"id": 3, "type": "group", "name": "things", "properties": [{"name": "recolour", "type": "string",
	   "value": "dark"}, {"name": "tint", "type": "color", "value": "#ff0000ff"}],
	   "layers": [{"id": 4, "type": "objectgroup", "name": "props", "objects": [{"id": 1, "gid": 1, "x": 0,
	    "y": 16, "width": 16, "height": 16, "properties": [{"name": "tint", "type": "color",
	    "value": "#ff808080"}]}]}]}]}`
	level, err := loadVariants(t, variantsFS(t, map[string]string{"variants.tmj": tmj}))
	require.NoError(t, err)

	floor, _ := level.TileLayer("floor")
	assert.Equal(t, darkRGB, cellPixel(floor, 0, 12, 2), "the layer's recolour")
	assert.Equal(t, stoneRGB, cellPixel(floor, 1, 0, 0), "a tile it holds no colour of stays as it is")
	assert.Equal(t, tiled(t, "#ff808080").Mul(tiled(t, "#ff00ff00")), floor.Tint(),
		"the layer's tint times Tiled's own")

	sky, _ := level.Layer("sky")
	assert.Equal(t, blackRGB, pixel(sky.Picture().Texture(), 12, 2), "an image layer's picture recoloured")

	things, _ := level.Layer("things")
	props, _ := things.Layer("props")
	s := level.Sprite(props.Objects()[0])
	assert.Equal(t, darkRGB, pixel(s.Texture(), 12, 2, s.Region().X, s.Region().Y), "its group's recolour")
	assert.Equal(t, tiled(t, "#ff0000ff").Mul(tiled(t, "#ff808080")), s.Tint(), "its group's tint times its own")
}

func TestAPaletteOrMaskTheLoaderDoesNotHoldIsRefusedWithTheNearest(t *testing.T) {
	tile := func(prop, value string) map[string]string {
		return map[string]string{"tiles.tsj": `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16,
		 "tiles": [{"id": 0, "properties": [{"name": "` + prop + `", "type": "string", "value": "` + value + `"}]}]}`}
	}
	layered := func(layer string) map[string]string {
		return map[string]string{"variants.tmj": `{"orientation": "orthogonal", "width": 1, "height": 1,
		 "tilewidth": 16, "tileheight": 16,
		 "tilesets": [{"firstgid": 1, "source": "tiles.tsj"}, {"firstgid": 3, "source": "masks.tsj"}],
		 "layers": [{"id": 1, "type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [1]},
		  ` + layer + `]}`}
	}
	object := func(gid, props string) map[string]string {
		return layered(`{"id": 2, "type": "objectgroup", "name": "props", "objects": [{"id": 1, "gid": ` + gid +
			`, "x": 0, "y": 16, "width": 16, "height": 16, "properties": [` + props + `]}]}`)
	}
	for _, c := range []struct {
		over map[string]string
		want string
	}{
		{tile("recolour", "drak"), "tile 0: property recolour: palette drak is not given; the nearest is dark"},
		{tile("mask", "hlaf"), "tile 0: property mask: mask hlaf is not in the mask tileset; the nearest is half"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "water", "width": 1, "height": 1, "data": [1],
		  "properties": [{"name": "recolour", "type": "string", "value": "deeep"}]}`),
			"layer water: property recolour: palette deeep is not given; the nearest is deep"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "water", "width": 1, "height": 1, "data": [1],
		  "properties": [{"name": "tint", "type": "int", "value": 3}]}`),
			"layer water: property tint: 3 is a int, not a color"},
		{object("1", `{"name": "recolour", "type": "string", "value": "dork"}`),
			"object 1 (): property recolour: palette dork is not given; the nearest is dark"},
		{object("1", `{"name": "mask", "type": "string", "value": "hold"}`),
			"object 1 (): property mask: mask hold is not in the mask tileset; the nearest is hole"},
		{object("0", `{"name": "tint", "type": "color", "value": "#ffff0000"}`),
			"object 1 (): tint, recolour and mask are a tile object's"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "cut", "class": "mask", "width": 1, "height": 1,
		  "data": [3], "properties": [{"name": "cuts", "type": "string", "value": "flor"}]}`),
			"layer cut: cuts flor: no layer is called so; the nearest is floor"},
		{layered(`{"id": 2, "type": "objectgroup", "name": "cut", "class": "mask", "objects": [],
		  "properties": [{"name": "cuts", "type": "string", "value": "floor"}]}`),
			"layer cut: a mask layer is a tile layer"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "cut", "class": "mask", "width": 1, "height": 1,
		  "data": [1], "properties": [{"name": "cuts", "type": "string", "value": "floor"}]}`),
			"layer cut: cell 0, 0: tile 0 has no name to find it in the mask tileset"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "cut", "class": "mask", "width": 1, "height": 1,
		  "data": [3]}`), "layer cut: property cuts: class mask requires it"},
		{layered(`{"id": 2, "type": "tilelayer", "name": "water", "width": 1, "height": 1, "data": [3],
		  "properties": [{"name": "recolour", "type": "string", "value": "deep"}]}`),
			"layer water: property recolour: palette (686f99>101010) matches no colour of the layer's tiles"},
	} {
		_, err := loadVariants(t, variantsFS(t, c.over))
		require.Error(t, err, c.want)
		assert.Contains(t, err.Error(), c.want)
	}

	_, err := LoadTiled(variantsFS(t, nil), "variants.tmj")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "palette dark: the loader is given no palettes (twod.Palettes)")
	_, err = LoadTiled(variantsFS(t, nil), "variants.tmj", Palettes(variantPalettes()))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mask hole: the loader is given no mask tileset (twod.Masks)")
}

func TestAMaskLayersCutStaysWithItsCellWhenTheGameSetsAnotherTile(t *testing.T) {
	level, err := loadVariants(t, os.DirFS("tiled/variants"))
	require.NoError(t, err)
	floor, _ := level.TileLayer("floor")

	floor.Set(2, 0, 1)
	assert.Equal(t, alpha(darkRGB, 0), cellPixel(floor, 2, 7, 7), "water now, recoloured and holed")
	assert.Equal(t, alpha(darkRGB, 0x80), cellPixel(floor, 2, 7, 3))
	assert.Equal(t, color.White, cellTint(floor, 2), "water has no tint of its own")

	floor.Set(0, 1, 2)
	assert.Equal(t, alpha(stoneRGB, 0), cellPixel(floor, 4, 12, 2), "stone, cut by the flipped half")
	floor.SetFlip(0, 1, TileFlipX)
	assert.Equal(t, alpha(stoneRGB, 0), cellPixel(floor, 4, 2, 2),
		"flipped as its mask is, the cut lies on the picture as it is")
	assert.Equal(t, stoneRGB, cellPixel(floor, 4, 12, 2))
}

func TestATallTilesTintIsItsSpritesTint(t *testing.T) {
	tsj := `{"image": "tiles.png", "tilewidth": 16, "tileheight": 16, "tiles": [{"id": 1, "properties": [
	 {"name": "tall", "type": "bool", "value": true}, {"name": "tint", "type": "color", "value": "#ff00ff00"}]}]}`
	tmj := `{"orientation": "orthogonal", "width": 1, "height": 1, "tilewidth": 16, "tileheight": 16,
	 "tilesets": [{"firstgid": 1, "source": "tiles.tsj"}],
	 "layers": [{"id": 1, "type": "tilelayer", "name": "floor", "width": 1, "height": 1, "data": [2],
	  "tintcolor": "#ff808080"}]}`
	fsys := variantsFS(t, map[string]string{"tiles.tsj": tsj, "variants.tmj": tmj})
	level, err := loadVariants(t, fsys)
	require.NoError(t, err)
	floor, _ := level.TileLayer("floor")
	require.Len(t, floor.TallSprites(), 1)
	assert.Equal(t, tiled(t, "#ff00ff00"), floor.TallSprites()[0].Tint(), "the map's tint reaches it as its child")

	standing := NewLayer(0, SortByY())
	level, err = loadVariants(t, fsys, TallIn(standing))
	require.NoError(t, err)
	floor, _ = level.TileLayer("floor")
	assert.Equal(t, tiled(t, "#ff00ff00").Mul(tiled(t, "#ff808080")), floor.TallSprites()[0].Tint(),
		"standing elsewhere, it takes the map's too")
}

func TestTheEnginesVariantPropertiesAreItsTileProperties(t *testing.T) {
	var names []string
	for _, d := range EngineTileProperties() {
		names = append(names, d.Name)
	}
	assert.Equal(t, "tall tint recolour mask", strings.Join(names, " "))
}
