package main

import (
	"embed"

	"github.com/lantervale/ember/twod"
)

//go:embed art/*.png
var art embed.FS

// Art is Flap's pixel art (Megacrash's, CC0, see art/CREDITS.md), drawn at
// 2 points a pixel.
type Art struct {
	birds, pipes, sky *twod.Texture
}

func loadArt() (*Art, error) {
	a := &Art{}
	for name, tex := range map[string]**twod.Texture{"birds": &a.birds, "pipes": &a.pipes, "sky": &a.sky} {
		t, err := twod.LoadTexture(art, "art/"+name+".png", twod.Pixelated())
		if err != nil {
			return nil, err
		}
		*tex = t
	}
	return a, nil
}
