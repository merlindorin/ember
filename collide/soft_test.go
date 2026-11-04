package collide //nolint:testpackage // white-box: the unexported crowd, its density and capacity

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullest is the most any crowd cell is covered, as a share of capacity.
func fullest(s *Space) float32 {
	var most float32
	for _, d := range s.crowd.density {
		most = max(most, d)
	}
	return most / capacity
}

func pile(n int, at geom.Vec2) []*thing {
	var out []*thing
	for range n {
		out = append(out, newThing(at.X, at.Y, NewBody(Box(16, 10), Soft(1))))
	}
	return out
}

func TestSoftBodiesStartedInOnePointSpreadUnderTheCap(t *testing.T) {
	things := pile(30, geom.V2(16, 21))
	w, space := setup(things...)
	space.crowd.rebuild(space)
	require.Greater(t, fullest(space), float32(4), "thirty on one point overfill their cell")
	w.RunFrames(200)
	assert.LessOrEqual(t, fullest(space), float32(1), "they spread until no cell is over capacity")
	for i, a := range things {
		for _, b := range things[i+1:] {
			assert.NotEqual(t, a.at, b.at, "no two stay on the same spot")
		}
	}
}

func TestSoftBodiesPassThroughEachOtherButStopAtHardOnes(t *testing.T) {
	a := newThing(0, 0, NewBody(Box(10, 10), Soft(1)))
	b := newThing(20, 0, NewBody(Box(10, 10), Soft(1)))
	wall := newThing(40, 0, NewBody(Box(10, 10)))
	w, space := setup(a, b, wall)
	w.RunFrames(1)
	moved, hits := space.Move(a, geom.V2(15, 0))
	assert.Equal(t, geom.V2(15, 0), moved, "a soft body moves into another soft one")
	assert.Empty(t, hits)
	moved, _ = space.Move(b, geom.V2(30, 0))
	assert.InDelta(t, 10, moved.X, 0.01, "and stops at a hard one")
	assert.True(t, NewBody(Box(1, 1), Soft(0.5)).Soft() == 0.5)
}

func TestASoftBodyMovesIntoACellOnlyWhileItHasRoom(t *testing.T) {
	crowd := pile(12, geom.V2(48, 16))
	walker := newThing(16, 16, NewBody(Box(16, 10), Soft(0.0001)))
	w, space := setup(append(crowd, walker)...)
	w.RunFrames(1)
	space.crowd.rebuild(space)
	full := space.crowd.slotOf(geom.V2(48, 11))
	require.Greater(t, space.crowd.density[full], float32(capacity))
	moved, _ := space.Move(walker, geom.V2(30, 0))
	assert.Equal(t, geom.Vec2{}, moved, "the full cell ahead stops it")
	moved, _ = space.Move(walker, geom.V2(0, 3))
	assert.Equal(t, geom.V2(0, 3), moved, "it still moves where there is room")
}

func TestSpreadingNeverPushesIntoSolidsOrTheHero(t *testing.T) {
	hero := newThing(40, 0, NewBody(Box(16, 16)))
	things := append(pile(60, geom.V2(16, 0)), hero)
	w, space := setup(things...)
	space.AddSolid(walls{{X: -40, Y: -200, W: 20, H: 400}, {X: 56, Y: -200, W: 20, H: 400}})
	for range 300 {
		for _, e := range things[:60] {
			space.Move(e, hero.at.Sub(e.at).Normalize())
		}
		w.RunFrames(1)
	}
	assert.Equal(t, geom.V2(40, 0), hero.at, "a crowd never pushes the hero")
	solids := walls{{X: -40, Y: -200, W: 20, H: 400}, {X: 56, Y: -200, W: 20, H: 400}, space.BodyOf(hero).Bounds()}
	for _, e := range things[:60] {
		box := space.BodyOf(e).Bounds()
		for _, r := range solids {
			assert.False(t, box.Overlaps(r), "a soft body at %v ends inside %v", e.at, r)
		}
	}
}

func TestTheSameStepsGiveTheSamePositions(t *testing.T) {
	run := func() []geom.Vec2 {
		h := newHorde(2000, 400, Soft(1))
		h.w.RunFrames(300)
		var out []geom.Vec2
		for _, t := range h.things {
			out = append(out, t.at)
		}
		return out
	}
	assert.Equal(t, run(), run())
}

func TestASettledHordeStaysBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("settles 20,000 bodies")
	}
	h := newHorde(20000, 1712, Soft(1))
	h.w.RunFrames(2000)
	assert.LessOrEqual(t, fullest(h.space), float32(1.5), "no cell of the pile over half again its capacity")
}

func TestABodyLeavingTheWorldLeavesTheCrowd(t *testing.T) {
	things := pile(10, geom.V2(0, 0))
	w, space := setup(things...)
	w.RunFrames(1)
	covered := func() float32 {
		var sum float32
		for _, d := range space.crowd.density {
			sum += d
		}
		return sum
	}
	before := covered()
	w.RemoveEntity(things[0])
	assert.InDelta(t, before-space.BodyOf(things[1]).share, covered(), 1e-4, "its share leaves at once")
	assert.Nil(t, space.BodyOf(things[0]))
	w.RunFrames(1)
	assert.InDelta(t, before-space.BodyOf(things[1]).share, covered(), 1e-4)
}
