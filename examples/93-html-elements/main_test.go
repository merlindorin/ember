package main

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/ui/align"
)

func TestElementsLineUp(t *testing.T) {
	for _, name := range names() {
		t.Run(name, func(t *testing.T) {
			e := elements()[name]
			doc := page(e)
			w, h := e.size(doc)
			doc.Layout(geom.V2(float32(w), float32(h)))
			for _, f := range align.Check(doc) {
				t.Error(f)
			}
			if len(doc.Warnings()) > 0 {
				t.Error(doc.Warnings())
			}
		})
	}
}
