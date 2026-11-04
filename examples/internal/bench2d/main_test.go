package main

import (
	"testing"

	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/twodgrid"
	"github.com/stretchr/testify/require"
)

// TestEveryBenchSceneDrawsTheSameWithGridsAndWithout builds each scene
// twice, one without twod's grids, steps both alike and compares what
// they emit, the window's list and every pass, frame by frame.
func TestEveryBenchSceneDrawsTheSameWithGridsAndWithout(t *testing.T) {
	scenes := []string{"wave23", "wide", "crowd", "20k", "200k"}
	if testing.Short() {
		scenes = scenes[:3]
	}
	defer func() { twodgrid.Off = false }()
	for _, scene := range scenes {
		t.Run(scene, func(t *testing.T) {
			twodgrid.Off = true
			plain, plainWin, err := build(scene, 1280, 720, &lightFlags{})
			require.NoError(t, err)
			twodgrid.Off = false
			fast, fastWin, err := build(scene, 1280, 720, &lightFlags{})
			require.NoError(t, err)
			for frame := range 40 {
				twodgrid.Off = true
				plain.Frame(1.0 / 50)
				twodgrid.Off = false
				fast.Frame(1.0 / 50)
				want, got := draw.FrontFor(plain, plainWin), draw.FrontFor(fast, fastWin)
				require.NotEmpty(t, want.Unindexed(), "frame %d", frame)
				require.Equal(t, want.Unindexed(), got.Unindexed(), "frame %d: the window's list", frame)
				require.Equal(t, batches(want.Batches), batches(got.Batches), "frame %d: its batches", frame)
				wp, gp := draw.PassesFor(plain, plainWin), draw.PassesFor(fast, fastWin)
				require.Len(t, gp, len(wp), "frame %d", frame)
				for i := range wp {
					require.Equal(t, wp[i].List.Unindexed(), gp[i].List.Unindexed(), "frame %d: pass %d", frame, i)
				}
			}
		})
	}
}

// batches are bs as compared across two Worlds: a batch's font by what it
// is, its metrics, not by its glyph cache, which fills as either World
// draws text (the vertices, UVs included, are compared apart).
func batches(bs []draw.Batch) []any {
	type font struct{ LineHeight, Ascent float32 }
	out := make([]any, len(bs))
	for i, b := range bs {
		var f *font
		if b.Atlas != nil {
			f = &font{b.Atlas.LineHeight(), b.Atlas.Ascent()}
		}
		b.Atlas = nil
		out[i] = struct {
			Batch draw.Batch
			Font  *font
		}{b, f}
	}
	return out
}
