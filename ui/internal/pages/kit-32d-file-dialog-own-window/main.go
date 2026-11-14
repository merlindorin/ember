// 32d FileDialog in its own window: kit.FileDialog as OwnWindow shows it, filling a
// 900×600 window without lights of its own, beach.png selected.
package main

import (
	_ "embed"
	"testing/fstest"
	"time"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	"github.com/lantervale/ember/ui/kit"
)

//go:embed beach.png
var beach []byte

func main() {
	at := time.Date(2026, 11, 2, 2, 54, 0, 0, time.UTC)
	file := func(size int) *fstest.MapFile { return &fstest.MapFile{Data: make([]byte, size), ModTime: at} }
	fsys := fstest.MapFS{
		"Pictures/Old/winter.jpg": file(840000),
		"Pictures/beach.png":      file(1200000),
		"Pictures/garden.jpg":     file(960000),
		"Pictures/list.txt":       file(523),
		"Pictures/sunset.png":     file(2400000),
		"Pictures/tree.png":       file(380000),
		"Music/song.ogg":          file(4100000),
		"Notes/ideas.md":          file(3000),
	}
	d := ui.NewDocument(kit.Theme(),
		ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte("html { background-color: #141210; }")},
			"beach.png": {Data: beach}}, "page.css"))
	fd := kit.NewFileDialog(d, fsys,
		kit.FileType("Pictures", ".png", ".jpg"),
		kit.RootName("Documents"),
		kit.Places("Pictures", "Music", "Notes"),
		kit.Recent("Pictures/beach.png", "Pictures/sunset.png", "Notes/ideas.md"),
		kit.DateFormat(func(t time.Time) string { return "Today, " + t.Format("15:04") }),
		kit.Preview(func(p string) string {
			if p == "Pictures/beach.png" {
				return "beach.png"
			}
			return ""
		}))
	fd.Open("Pictures")
	fd.Select("Pictures/beach.png")
	fd.Node().AddClass("kit-own")
	pages.Run(d, 900, 600)
}
