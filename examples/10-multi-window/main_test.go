package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/camera"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/ui"
)

func TestEachWindowHasItsCameraAndUI(t *testing.T) {
	sceneWindow, mapWindow := newSceneWindow(), newMapWindow()
	w := ember.New().AddEntities(sceneWindow, mapWindow)

	var orbit *camera.OrbitCamera
	var eye *Eye
	var label *ui.Document
	for _, e := range w.Entities() {
		switch e := e.(type) {
		case *camera.OrbitCamera:
			orbit = e
		case *Eye:
			eye = e
		case *ui.Document:
			if e.Body().Text() == "map" {
				label = e
			}
		}
	}
	if orbit == nil || eye == nil || label == nil {
		t.Fatal("each window brings its camera and its label")
	}

	w.RunFrames(3)
	if orbit.RendersIn() != sceneWindow || eye.RendersIn() != mapWindow {
		t.Fatal("a camera draws in the window it is a child of")
	}
	if draw.FrontFor(w, nil) == nil || draw.FrontFor(w, mapWindow) == nil {
		t.Fatal("both windows get a UI")
	}

	w.RemoveEntity(mapWindow)
	if w.Has(eye) || w.Has(label) {
		t.Fatal("closing the map window takes its camera and label with it")
	}
}
