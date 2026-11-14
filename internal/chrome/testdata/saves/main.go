//go:build js

// Command saves counts its runs in a file under the user's config folder,
// written as the demo writes its saves (MkdirAll, a temporary file,
// Rename), and publishes the count as window.emberSaves; with EMBER_PANIC
// set it panics in its first frame, for the page's crash overlay.
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall/js"

	"github.com/lantervale/ember"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type panicker struct{}

func (panicker) Update(world.Frame) { panic("the probe was asked to panic") }

func main() {
	dir, err := os.UserConfigDir()
	if err != nil {
		js.Global().Set("emberSaves", "error: "+err.Error())
		return
	}
	path := filepath.Join(dir, "ember-saves", "runs.txt")
	n := 0
	if data, err := os.ReadFile(path); err == nil {
		n, _ = strconv.Atoi(string(data))
	}
	n++
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		js.Global().Set("emberSaves", "error: "+err.Error())
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(n)), 0o644); err != nil {
		js.Global().Set("emberSaves", "error: "+err.Error())
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		js.Global().Set("emberSaves", "error: "+err.Error())
		return
	}
	js.Global().Set("emberSaves", strconv.Itoa(n))
	w := ember.New().AddEntities(platform.NewWindow(platform.Size(160, 100)))
	if os.Getenv("EMBER_PANIC") != "" {
		w.AddSystems(panicker{})
	}
	w.MustRun()
}
