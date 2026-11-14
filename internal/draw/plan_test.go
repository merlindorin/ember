package draw_test

import (
	"slices"
	"testing"

	"github.com/lantervale/ember/internal/draw"
)

const (
	floorClass = 1 << draw.OccluderGroups
	wallsClass = 2 << draw.OccluderGroups
	innerClass = 3 << draw.OccluderGroups
)

// affected is a plan's lights, one lighting the floor, one the walls, and
// its classes, the floor, the walls and a layer nested in the walls.
func affected() ([]draw.Light, [][]any) {
	floor, walls, inner := "floor", "walls", "inner"
	return []draw.Light{{}, {Affects: []any{floor}}, {Affects: []any{walls}}, {Kind: draw.DirectionalLight}},
		[][]any{{floor}, {walls}, {inner, walls}}
}

func samePictures(a, b []draw.Picture) bool {
	return slices.EqualFunc(
		a,
		b,
		func(a, b draw.Picture) bool { return a.Left == b.Left && slices.Equal(a.Excluded, b.Excluded) },
	)
}

func TestAPlanGivesEachSetOfLightsLeftOutAPicture(t *testing.T) {
	lights, classes := affected()
	var p draw.PicturePlan
	p.Plan(lights, classes, draw.KeysOf(0, floorClass, wallsClass, innerClass, wallsClass|0b100))
	want := []draw.Picture{
		{Excluded: []int{1, 2}},
		{Excluded: []int{2}},
		{Excluded: []int{1}},
		{Left: 0b100, Excluded: []int{1}},
	}
	if !samePictures(p.Pictures, want) {
		t.Fatalf(
			"under no named layer both lamps are left out, under the floor the walls' lamp, under the walls the floor's: %+v",
			p.Pictures,
		)
	}
	if p.Of[0] != 1 || p.Of[floorClass] != 2 || p.Of[wallsClass] != 3 || p.Of[wallsClass|0b100] != 4 {
		t.Fatalf("each key reads its picture: %v", p.Of[:24])
	}
	if p.Of[innerClass] != 3 {
		t.Fatal("a layer nested in the walls is lit by the walls' lamp: the walls' picture")
	}
	if len(p.Fallen) != 0 {
		t.Fatalf("all fit: %v", p.Fallen)
	}
}

func TestPastFourPicturesAKeyFallsBackToTheWindowsPicture(t *testing.T) {
	lights, classes := affected()
	var p draw.PicturePlan
	p.Plan(lights, classes, draw.KeysOf(0, 0b100, floorClass, floorClass|0b100, wallsClass, innerClass))
	if len(p.Pictures) != draw.MaxPictures {
		t.Fatalf("four pictures: %+v", p.Pictures)
	}
	if !slices.Equal(p.Fallen, []int{wallsClass, innerClass}) || p.Of[wallsClass] != 0 || p.Of[innerClass] != 0 {
		t.Fatalf("the keys past them read the window's picture: %v", p.Fallen)
	}
}

func TestAPlanWithoutAffectsIsShadedByAlone(t *testing.T) {
	var p draw.PicturePlan
	p.Plan([]draw.Light{{}, {}}, nil, draw.KeysOf(0, 0b100))
	if !samePictures(p.Pictures, []draw.Picture{{Left: 0b100}}) || p.Of[0] != 0 || p.Of[0b100] != 1 {
		t.Fatalf("one picture for the groups left out, none for key 0: %+v", p)
	}
	p.Plan([]draw.Light{{}, {}}, nil, draw.Keys{})
	if len(p.Pictures) != 0 {
		t.Fatal("nothing shaded, no picture")
	}
}
