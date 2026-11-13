// Package crypt is art and sound for the examples' benchmarks: sprites
// from 0x72's 16x16 DungeonTileset II, the helpers that cut them into
// tilesets and animations, and the music and sound effects.
package crypt

import (
	"embed"
	"fmt"
	"image"
	"image/draw"
	_ "image/png" // registers the PNG decoder that image.Decode uses
	"io/fs"
	"strings"

	"github.com/merlindorin/ember/collide/tiles"
	"github.com/merlindorin/ember/twod"
)

//go:embed art/*.png
var art embed.FS

//go:generate go run sounds_gen.go

//go:embed sound/*.ogg
var sound embed.FS

// Sounds are a music track, music.ogg, and sound effects:
// shoot, hit, kill, powerup, hurt and gameover, each an .ogg.
func Sounds() fs.FS {
	sub, _ := fs.Sub(sound, "sound")
	return sub
}

// The floor's tiles: eight stone variants, then a skull lying on stone.
const (
	Stone = 1
	Skull = 9
)

// The things standing on the floor: a crate and a column, both solid.
const (
	Crate = iota + 1
	Column
)

// Floor is the crypt's floor, a tile of 16 pixels for each stone and the
// skull.
func Floor() *twod.Tileset {
	names := []string{
		"floor_1",
		"floor_2",
		"floor_3",
		"floor_4",
		"floor_5",
		"floor_6",
		"floor_7",
		"floor_8",
		"floor_1+skull",
	}
	return twod.NewTileset(strip(16, 16, names...), twod.Cell(16, 16))
}

// Things are what stands on the floor and stops whoever walks into it,
// drawn tall so characters pass in front of and behind them.
func Things() *twod.Tileset {
	return twod.NewTileset(
		strip(16, 48, "crate", "column"),
		twod.Cell(16, 48),
		twod.Tall(Crate, Column),
		tiles.Solid(Crate, Column),
	)
}

// Character is the sheet of a hero or a monster, such as "elf_m" or
// "goblin", and its two clips, "idle" and "run", four frames each, facing
// right.
func Character(name string) (*twod.Texture, *twod.Animations) {
	var names []string
	for _, clip := range []string{"idle", "run"} {
		for i := range 4 {
			names = append(names, fmt.Sprintf("%s_%s_anim_f%d", name, clip, i))
		}
	}
	w, h := size(names[0])
	sheet := strip(w, h, names...)
	frames := twod.Strip(sheet, twod.Cell(w, h))
	return sheet, twod.NewAnimations().
		Add("idle", 6, frames[0:4], twod.Loop()).
		Add("run", 10, frames[4:8], twod.Loop())
}

// Sprite is one picture, such as "weapon_bow" or "flask_red".
func Sprite(name string) *twod.Texture { return strip(0, 0, name) }

// strip puts the pictures side by side in cells of w by h pixels, each
// standing on its cell's bottom edge; w 0 is the picture's own size. A name
// such as "floor_1+skull" lays the second picture over the first.
func strip(w, h int, names ...string) *twod.Texture {
	if w == 0 {
		w, h = size(names[0])
	}
	sheet := image.NewNRGBA(image.Rect(0, 0, w*len(names), h))
	for i, n := range names {
		for _, layer := range strings.Split(n, "+") {
			img := load(layer)
			b := img.Bounds()
			at := image.Pt(i*w+(w-b.Dx())/2, h-b.Dy())
			draw.Draw(sheet, image.Rectangle{Min: at, Max: at.Add(b.Size())}, img, b.Min, draw.Over)
		}
	}
	return twod.NewTexture(sheet, twod.Pixelated())
}

func size(name string) (int, int) {
	b := load(name).Bounds()
	return b.Dx(), b.Dy()
}

func load(name string) image.Image {
	f, err := art.Open("art/" + name + ".png")
	if err != nil {
		panic("crypt: no picture " + name)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic("crypt: " + name + ": " + err.Error())
	}
	return img
}
