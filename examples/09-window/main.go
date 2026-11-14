// A window: an entity made with options. This one can't be resized below
// 320×180 and keeps a 16:9 shape. The label's Document, the ground and the
// camera are its children, so they show in it. ShowSize writes the window's size in
// the label, and F toggles fullscreen.
//
//	go run ./examples/09-window
package main

import (
	"github.com/lantervale/ember"
	"github.com/lantervale/ember/camera"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
)

func main() {
	win := platform.NewWindow(
		platform.Title("ember · window"),
		platform.Size(640, 360),
		platform.MinSize(320, 180),
		platform.AspectRatio(16, 9),
		platform.Background(color.Hex("#2b2f3a")),
	)

	size := kit.Label("")
	label := ui.NewDocument(kit.Theme(), ui.CSS(`body { padding: 16px; }`))
	label.Body().Append(size)
	eye := camera.NewOrbit(
		camera.Distance(11),
		camera.Angles(0.64, 0.46),
	)
	win.Add(
		label,
		newGround(),
		eye,
	)

	ember.New().
		AddEntities(win).
		AddSystems(
			&ShowSize{window: win, label: size},
			&ToggleFullscreen{window: win},
		).
		MustRun()
}
