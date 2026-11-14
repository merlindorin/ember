package main

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/align"
)

func TestElementsLineUp(t *testing.T) {
	for _, name := range names() {
		t.Run(name, func(t *testing.T) {
			g, err := page(elements()[name])
			if err != nil {
				t.Fatal(err)
			}
			g.doc.Layout(geom.V2(480, 320))
			for _, f := range align.Check(g.doc) {
				t.Error(f)
			}
		})
	}
}
