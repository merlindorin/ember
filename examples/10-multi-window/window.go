package main

import (
	"github.com/lantervale/ember/camera"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/render"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/kit"
	"github.com/lantervale/ember/world"
)

// newSceneWindow is the primary window, added first, with its label and
// an orbit camera, which reads this window's mouse and draws in it because
// it is the window's child.
func newSceneWindow() *platform.Window {
	w := platform.NewWindow(
		platform.Title("ember · scene"),
		platform.Size(900, 600),
		platform.Position(60, 80),
	)
	w.Add(
		camera.NewOrbit(camera.Distance(11), camera.Angles(0.64, 0.46)),
		newLabel("scene"),
	)
	return w
}

// newMapWindow opens a second window with its own camera, looking down on
// the scene, and its own label. Both are the window's children, so they
// show in it and leave with it.
func newMapWindow() *platform.Window {
	w := platform.NewWindow(
		platform.Title("ember · map"),
		platform.Size(400, 400),
		platform.Position(980, 80),
	)
	w.Add(
		&Eye{world.NewBasic(), scene.At(0, 20, 0.01).LookingAt(geom.Zero3), render.Camera{OrthoHeight: 12}},
		newLabel("map"),
	)
	return w
}

// newLabel is a window's label: a kit Label in a Document of its own,
// padded 16 points from the corner.
func newLabel(text string) *ui.Document {
	doc := ui.NewDocument(kit.Theme(), ui.CSS(`body { padding: 16px; }`))
	doc.Body().Append(kit.Label(text))
	return doc
}
