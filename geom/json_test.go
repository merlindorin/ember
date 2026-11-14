package geom_test

import (
	"encoding/json"
	"testing"

	"github.com/lantervale/ember/geom"
)

func TestGeomValuesMarshalByTheirFieldNames(t *testing.T) {
	for _, c := range []struct {
		v    any
		want string
	}{
		{geom.V2(1, 2), `{"X":1,"Y":2}`},
		{geom.V3(1, 2, 3), `{"X":1,"Y":2,"Z":3}`},
		{geom.V4(1, 2, 3, 4), `{"X":1,"Y":2,"Z":3,"W":4}`},
		{geom.Rect{X: 1, Y: 2, W: 3, H: 4}, `{"X":1,"Y":2,"W":3,"H":4}`},
		{geom.Quat{X: 1, Y: 2, Z: 3, W: 4}, `{"X":1,"Y":2,"Z":3,"W":4}`},
		{
			geom.Transform2D{X: geom.V2(1, 0), Y: geom.V2(0, 1), Origin: geom.V2(5, 6)},
			`{"X":{"X":1,"Y":0},"Y":{"X":0,"Y":1},"Origin":{"X":5,"Y":6}}`,
		},
		{geom.Circle{Center: geom.V2(1, 2), Radius: 3}, `{"Center":{"X":1,"Y":2},"Radius":3}`},
		{
			geom.Ray{Origin: geom.V3(1, 2, 3), Dir: geom.V3(0, 0, 1)},
			`{"Origin":{"X":1,"Y":2,"Z":3},"Dir":{"X":0,"Y":0,"Z":1}}`,
		},
		{geom.AABB{Min: geom.V3(0, 0, 0), Max: geom.V3(1, 1, 1)}, `{"Min":{"X":0,"Y":0,"Z":0},"Max":{"X":1,"Y":1,"Z":1}}`},
	} {
		b, err := json.Marshal(c.v)
		if err != nil || string(b) != c.want {
			t.Fatalf("%T: %s, want %s (%v)", c.v, b, c.want, err)
		}
	}
}
