// Package pages runs a ui test page: in a window, which ember's capture
// pictures, or, with EMBER_UI_EXPORT set to a file, written there as HTML for
// Chrome, its canvases drawn from the capture EMBER_UI_SNAPSHOTS names.
// internal/chrome's pixel tests run both and compare them. With
// EMBER_UI_BENCH set, it measures the page's frames instead (BenchmarkKitPages),
// and with EMBER_UI_ALIGN set to a file it writes there, as JSON, what
// align finds out of line (kit's TestKitPagesLineUp).
package pages

import (
	"encoding/json"
	"os"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/ui/export"
	"github.com/merlindorin/ember/ui/internal/assets"
	"github.com/merlindorin/ember/ui/internal/dom"
)

// Run shows d in a window of w×h points, or exports it laid out at that
// size over its zoom, as the window lays it out, so export can draw what
// needs a size, such as a range.
func Run(d *ui.Document, w, h int) {
	if os.Getenv("EMBER_UI_BENCH") != "" {
		bench(d, w, h)
		return
	}
	if path := os.Getenv("EMBER_UI_ALIGN"); path != "" {
		d.Layout(geom.V2(float32(w), float32(h)).Scale(1 / d.Zoom()))
		data, err := json.Marshal(align.Check(d))
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile(path, data, 0o600); err != nil { //nolint:gosec // the harness's own path, from EMBER_UI_ALIGN
			panic(err)
		}
		return
	}
	if path := os.Getenv("EMBER_UI_EXPORT"); path != "" {
		f, err := os.Create(path) //nolint:gosec // the path is the test harness's own, from EMBER_UI_EXPORT
		if err != nil {
			panic(err)
		}
		defer f.Close()
		d.Layout(geom.V2(float32(w), float32(h)).Scale(1 / d.Zoom()))
		var opts []export.Option
		if shot := os.Getenv("EMBER_UI_SNAPSHOTS"); shot != "" {
			opts = append(opts, snapshots(d, shot, w))
		}
		if err = export.Write(f, d, opts...); err != nil {
			panic(err)
		}
		return
	}
	win := platform.NewWindow(platform.Size(w, h), platform.FixedSize(), platform.Background(color.Hex("#ffffff")))
	win.Add(d)
	ember.New().AddEntities(win).MustRun()
}

// Pin holds pseudo-classes on n, such as dom.Hover|dom.FocusVisible, so
// the page's picture shows that state in ember and in Chrome alike.
func Pin(d *ui.Document, n *ui.Node, s dom.State) { assets.Pin(d, n, s) }
