package main

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
)

// shadowShapes are tile id's shapes that block light only, of class shadow,
// as light reads them.
func shadowShapes(ts *twod.Tileset, id int) []geom.Rect {
	var out []geom.Rect
	for _, s := range ts.Shapes(id) {
		if s.Class == "shadow" {
			out = append(out, s.Rect)
		}
	}
	return out
}

// casts reports whether tile id's shadow property lets it cast.
func casts(ts *twod.Tileset, id int) bool {
	v, _ := ts.Value(id, "shadow")
	return v.String() != "none"
}
