package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/collide"
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestALightNoOccluderReachesIsUnblocked(t *testing.T) {
	w, win, _ := scene(t)
	pillar := twod.NewSprite(picture(4, 4), twod.At(-20, 0), Occluder())
	near := NewPoint(50, color.White, At(0, 0))
	other := NewPoint(50, color.White, At(0, 10), BlockedBy(2))
	far := NewPoint(30, color.White, At(150, 100))
	xray := NewPoint(50, color.White, At(0, -10), Shadows(false))
	layer := twod.NewLayer(0, Lit()).Add(pillar, near, other, far, xray)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 4)
	assert.False(t, unblocked(l, 400, 300), "the pillar stands within its reach")
	assert.True(t, unblocked(l, 400, 320), "the pillar is not of a group it is blocked by")
	assert.True(t, unblocked(l, 700, 500), "nothing stands within its reach")
	assert.False(t, unblocked(l, 400, 280), "a light casting no shadow is left as it is")
}

func TestSolidsAndSilhouettesAsBlockers(t *testing.T) {
	space := collide.NewSpace()
	space.AddSolid(walls{{X: 100, Y: -10, W: 20, H: 20}})
	w, win, _ := scene(t, OccludersFrom(space))
	hero := twod.NewSprite(picture(10, 20), twod.At(-100, 0), Silhouette())
	byWall := NewPoint(40, color.White, At(80, 0))
	byHero := NewPoint(40, color.White, At(-80, 0))
	layer := twod.NewLayer(0, Lit()).Add(hero, byWall, byHero)
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.Len(t, l.Lights, 2)
	assert.False(t, unblocked(l, 560, 300), "a solid of the space is an occluder")
	assert.True(t, unblocked(l, 240, 300), "a silhouette casts its own shadow, never a shadow row's")
}

// unblocked is whether l's light at x, y in window points is Unblocked.
func unblocked(l *draw.Lighting, x, y float32) bool {
	for _, li := range l.Lights {
		if li.At.Sub(geom.V2(x, y)).Len() < 0.5 {
			return li.Unblocked
		}
	}
	panic("no light there")
}

func TestABlockedGridCoversWhatItWasGiven(t *testing.T) {
	var b blocked
	b.reset(geom.Rect{X: -10, Y: -10, W: 300, H: 200})
	b.add(geom.Rect{X: 130, Y: 70, W: 4, H: 4}, 2)
	b.add(geom.Rect{X: -50, Y: -50, W: 10, H: 10}, 1)
	assert.True(t, b.any(geom.Rect{X: 100, Y: 50, W: 30, H: 30}, 2))
	assert.False(t, b.any(geom.Rect{X: 100, Y: 50, W: 30, H: 30}, 1), "of another group")
	assert.False(t, b.any(geom.Rect{X: 200, Y: 120, W: 30, H: 30}, 3), "cells away")
	assert.False(t, b.any(geom.Rect{X: -10, Y: -10, W: 10, H: 10}, 1), "off the grid marks nothing")
}
