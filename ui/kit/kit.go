// Package kit is ember's components on ui: each is an element with a
// class, or a small composite where HTML has none, styled by the shared
// sheet Theme gives a Document. A component lives in its own file here,
// its CSS in css/ under the same number and name.
package kit

import (
	"embed"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing/fstest"

	"github.com/merlindorin/ember/ui"
)

//go:embed css/*.css images/*.png
var sheets embed.FS

// Theme is ember's look for a Document: the palette as CSS variables
// named as the old ember/ui's theme names them, then each component's rules.
func Theme() ui.DocumentOption {
	return ui.Stylesheet(themeFS(), "ember.css")
}

// themeFS is one sheet, ember.css, made of css/*.css in name order, and
// the pictures it names, under images/.
func themeFS() fs.FS {
	entries, err := fs.ReadDir(sheets, "css")
	if err != nil {
		panic(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	var b strings.Builder
	for _, n := range names {
		var data []byte
		data, err = fs.ReadFile(sheets, path.Join("css", n))
		if err != nil {
			panic(err)
		}
		b.WriteString("/* " + n + " */\n")
		b.Write(data)
		b.WriteByte('\n')
	}
	files := fstest.MapFS{"ember.css": {Data: []byte(b.String())}}
	pictures, err := fs.ReadDir(sheets, "images")
	if err != nil {
		panic(err)
	}
	for _, e := range pictures {
		name := path.Join("images", e.Name())
		var data []byte
		data, err = fs.ReadFile(sheets, name)
		if err != nil {
			panic(err)
		}
		files[name] = &fstest.MapFile{Data: data}
	}
	return files
}
