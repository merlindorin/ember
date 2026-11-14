package twod //nolint:testpackage // white-box: draws into the canvas's own list, empties the variants' cache

import (
	"encoding/json"
	"image"
	"os"
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

// BenchmarkTileMapDrawVariants is BenchmarkTileMapDraw's map with every
// cell drawn through a look made at load: recoloured by its layer and cut
// by a mask layer. Beside BenchmarkTileMapDraw, it is what a frame pays
// for variants: nothing but the look's lookup.
func BenchmarkTileMapDrawVariants(b *testing.B) {
	w, _, _ := app()
	c := CanvasFor(w, nil)
	m := drawBenchMap()
	pal := NewPalette(Swap(color.RGB8(0x68, 0x6f, 0x99), color.RGB8(0x2c, 0x35, 0x4d)))
	if err := m.setRecolour(pal); err != nil {
		b.Fatal(err)
	}
	mask := NewTexture(image.NewNRGBA(image.Rect(0, 0, 16, 16)))
	for i := range m.cells {
		if err := m.cut(i, maskCut{tex: mask, rect: geom.Rect{W: 16, H: 16}}); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		c.list.Reset(0)
		m.Draw(c)
	}
}

// BenchmarkLoadTiledVariants is a load of tiled/variants: plain, its
// tint, recolour and mask properties left out; cold, every variant made
// anew; warm, made by an earlier load, as a game reloading a level finds
// them. The mask tileset is loaded once, as a game does.
func BenchmarkLoadTiledVariants(b *testing.B) {
	data, readErr := os.ReadFile("tiled/variants/variants.tmj")
	if readErr != nil {
		b.Fatal(readErr)
	}
	plain := variantsFS(b, map[string]string{"variants.tmj": withoutVariants(string(data))})
	all := variantsFS(b, nil)
	masks, maskErr := LoadTiledTileset(all, "masks.tsj")
	if maskErr != nil {
		b.Fatal(maskErr)
	}
	opts := []TileMapOption{Palettes(variantPalettes()), Masks(masks)}
	b.Run("plain", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := LoadTiled(plain, "variants.tmj"); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("cold", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			variants.Lock()
			variants.made = map[variantKey]*Texture{}
			variants.Unlock()
			if _, err := LoadTiled(all, "variants.tmj", opts...); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("warm", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := LoadTiled(all, "variants.tmj", opts...); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// withoutVariants is map tmj without its mask layers and its layers' and
// objects' properties.
func withoutVariants(tmj string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(tmj), &m); err != nil {
		panic(err)
	}
	var layers []any
	all, _ := m["layers"].([]any)
	for _, l := range all {
		layer, _ := l.(map[string]any)
		if layer["class"] == maskClass {
			continue
		}
		delete(layer, "properties")
		objects, _ := layer["objects"].([]any)
		for _, o := range objects {
			object, _ := o.(map[string]any)
			delete(object, "properties")
		}
		layers = append(layers, layer)
	}
	m["layers"] = layers
	out, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(out)
}
