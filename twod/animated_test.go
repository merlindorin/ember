package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"encoding/json"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func walker() *Sprite {
	sheet := atlas(48, 16)
	anims := NewAnimations().
		Add("walk", 10, Strip(sheet, Cell(16, 16)), Loop()).
		Add("die", 10, Strip(sheet, Cell(16, 16))[:2], Once())
	return NewSprite(nil, Animate(anims))
}

func TestStripCutsASheetLeftToRightThenDown(t *testing.T) {
	frames := Strip(atlas(32, 16), Cell(16, 8))
	require.Len(t, frames, 4)
	assert.Equal(t, geom.Rect{X: 16, Y: 0, W: 16, H: 8}, frames[1].Region)
	assert.Equal(t, geom.Rect{X: 0, Y: 8, W: 16, H: 8}, frames[2].Region)
}

func TestPictureIsTheFrameShownNow(t *testing.T) {
	s := walker()
	s.Play("walk")
	s.step(0.25)
	tex, region := s.Picture()
	clip := s.Animations().clip("walk")
	assert.Equal(t, clip.frames[2].Texture, tex)
	assert.Equal(t, clip.frames[2].Region, region, "the walk's third frame")
	still := NewSprite(atlas(8, 4))
	_, region = still.Picture()
	assert.Equal(t, geom.Rect{W: 8, H: 4}, region, "a still sprite shows its whole texture")
}

func TestAnimatedSpritesStepAndRaiseEvents(t *testing.T) {
	s := walker()
	if name, playing := s.Playing(); name != "walk" || playing {
		t.Fatal("stopped on the first clip until played")
	}
	assert.Empty(t, s.step(1), "nothing moves while stopped")
	s.Play("walk")
	assert.Empty(t, s.step(0.25))
	assert.Equal(t, 2, s.Frame())
	assert.Equal(t, []any{AnimationLooped{Sprite: s, Name: "walk"}}, s.step(0.1), "it loops")
	assert.Equal(t, 0, s.Frame())

	Speed(2).sprite(s)
	s.step(0.05)
	assert.Equal(t, 1, s.Frame(), "twice the speed")

	s.Play("die")
	assert.Equal(t, 0, s.Frame(), "a new clip starts over")
	assert.Equal(t, []any{AnimationFinished{Sprite: s, Name: "die"}}, s.step(0.1))
	assert.Equal(t, 1, s.Frame(), "a clip played once stays on its last frame")
	_, playing := s.Playing()
	assert.False(t, playing)
	assert.Empty(t, s.step(1), "and finishes once")

	s.Play("walk")
	s.step(0.05)
	s.Play("walk")
	assert.Equal(t, 1, s.Frame(), "playing the clip that plays keeps its place")
	s.Stop()
	assert.Equal(t, 0, s.Frame())
}

func TestTheSystemPlaysAnimationsAndEmitsTheirEvents(t *testing.T) {
	w, win, c := app()
	s := walker()
	Playing("die").sprite(s)
	win.AppendChild(s)
	w.AddEntities(s)
	var finished world.Reader[AnimationFinished]
	var got []AnimationFinished
	w.On(world.Update, "read", func(f world.Frame) {
		for e := range finished.Read(f.World) {
			got = append(got, e)
		}
	})
	for range 30 {
		w.Tick(1.0 / 60)
	}
	assert.Equal(t, []AnimationFinished{{Sprite: s, Name: "die"}}, got)
	require.Len(t, c.Unindexed(), 6)
	assert.Equal(t, geom.V2(16.0/48, 0), c.Unindexed()[0].UV, "it shows the clip's last frame")
}

func TestAPingPongClipPlaysForthAndBack(t *testing.T) {
	sheet := atlas(48, 16)
	s := NewSprite(nil, Animate(NewAnimations().Add("bob", 10, Strip(sheet, Cell(16, 16)), PingPong())), Playing("bob"))
	var frames []int
	looped := 0
	for range 6 {
		for _, ev := range s.step(0.1) {
			if _, ok := ev.(AnimationLooped); ok {
				looped++
			}
		}
		frames = append(frames, s.Frame())
	}
	assert.Equal(t, []int{1, 2, 1, 0, 1, 2}, frames)
	assert.Equal(t, 1, looped, "back at the start, it has looped")
}

func TestClipsSaveTheirRepeatAsAWord(t *testing.T) {
	sheet := atlas(32, 16)
	a := NewAnimations().Add("bob", 8, Strip(sheet, Cell(16, 16)), PingPong())
	data, err := json.Marshal(a)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"repeat":"ping_pong"`)
	var back Animations
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, pingPong, back.clip("bob").repeat)

	old := []byte(`{"textures":[],"clips":[{"name":"walk","fps":8,"loop":true,"frames":[]}]}`)
	require.NoError(t, json.Unmarshal(old, &back))
	assert.Equal(t, loop, back.clip("walk").repeat, "the old loop flag still reads")
}

func TestCellIsAGridOfPixelCells(t *testing.T) {
	w, h := Cell(16, 8).Size()
	assert.Equal(t, 16, w)
	assert.Equal(t, 8, h)
	assert.Len(t, Strip(atlas(64, 48), Cell(16, 16)), 12)
}

func TestOnlyPlayingSpritesAreStepped(t *testing.T) {
	w, win, _ := app()
	sheet := atlas(64, 16)
	clips := NewAnimations().Add("walk", 10, Strip(sheet, Cell(16, 16)), Loop())
	walker := NewSprite(nil, Animate(clips), Playing("walk"))
	stopped := NewSprite(nil, Animate(clips), Playing("walk"))
	still := NewSprite(sheet)
	win.Add(walker, stopped, still)
	var sys *System
	for _, s := range w.Systems() {
		if x, ok := s.(*System); ok {
			sys = x
		}
	}
	w.Frame(1.0 / 60)
	stopped.SetFrame(1)
	stopped.Pause()
	for range 30 {
		w.Frame(1.0 / 60)
	}
	if walker.Frame() == 0 {
		t.Fatal("a playing clip advances")
	}
	if stopped.Frame() != 1 {
		t.Fatalf("a paused clip holds its frame: %d", stopped.Frame())
	}
	for _, p := range sys.players.list {
		if p.sprite == still || p.sprite == stopped {
			t.Fatal("only playing sprites are listed for stepping")
		}
	}
	stopped.Play("walk")
	w.RemoveEntity(walker)
	win.Add(walker)
	w.Frame(1.0 / 60)
	listed := map[*Sprite]int{}
	for _, p := range sys.players.list {
		if p.turn == p.sprite.turn {
			listed[p.sprite]++
		}
	}
	if listed[stopped] != 1 || listed[walker] != 1 {
		t.Fatalf("played again and added again, each listed once: %v", listed)
	}
}
