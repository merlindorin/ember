// Package lucide is the Lucide icon font for the examples, read into a
// ui kit icon set. Lucide is under the ISC licence; see LICENSE and CREDITS.md.
package lucide

import (
	"embed"

	"github.com/merlindorin/ember/ui/kit"
)

//go:embed lucide.ttf codepoints.json
var files embed.FS

// Kit reads the font and its names into a kit icon set: "sun", "play",
// "eye"… as in lucide.dev/icons.
func Kit() (*kit.Icons, error) { return kit.LoadIcons(files, "lucide.ttf", "codepoints.json") }
