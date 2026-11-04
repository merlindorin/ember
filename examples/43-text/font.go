package main

import (
	"embed"

	"github.com/merlindorin/ember/twod"
)

//go:embed fonts
var files embed.FS

// loadPixelFont reads Silkscreen, a pixel font under the SIL Open Font
// Licence, its licence next to it. Any TrueType font loads the same way.
func loadPixelFont() (*twod.Face, error) {
	return twod.LoadFace(files, "fonts/Silkscreen-Regular.ttf")
}
