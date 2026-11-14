package ease_test

import (
	"testing"

	"github.com/lantervale/ember/ease"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryEasingRunsFromZeroToOne(t *testing.T) {
	all := map[string]ease.Func{
		"Linear": ease.Linear,
		"InQuad": ease.InQuad, "OutQuad": ease.OutQuad, "InOutQuad": ease.InOutQuad,
		"InCubic": ease.InCubic, "OutCubic": ease.OutCubic, "InOutCubic": ease.InOutCubic,
		"InSine": ease.InSine, "OutSine": ease.OutSine, "InOutSine": ease.InOutSine,
		"InExpo": ease.InExpo, "OutExpo": ease.OutExpo, "InOutExpo": ease.InOutExpo,
		"InBack": ease.InBack, "OutBack": ease.OutBack, "InOutBack": ease.InOutBack,
		"InElastic": ease.InElastic, "OutElastic": ease.OutElastic, "InOutElastic": ease.InOutElastic,
		"InBounce": ease.InBounce, "OutBounce": ease.OutBounce, "InOutBounce": ease.InOutBounce,
	}
	for name, f := range all {
		assert.InDelta(t, 0, f(0), 1e-3, name)
		assert.InDelta(t, 1, f(1), 1e-3, name)
	}
}

func TestEasingsHaveTheirShape(t *testing.T) {
	assert.Less(t, ease.InQuad(0.5), float32(0.5), "In starts slow")
	assert.Greater(t, ease.OutQuad(0.5), float32(0.5), "Out starts fast")
	assert.InDelta(t, 0.5, ease.InOutCubic(0.5), 1e-5, "InOut is halfway at half time")
	assert.InDelta(t, 0.5, ease.InOutSine(0.5), 1e-5)
	assert.Less(t, ease.InBack(0.2), float32(0), "Back pulls back first")
	assert.Greater(t, ease.OutBack(0.8), float32(1), "and overshoots at the end")
	assert.Greater(t, ease.OutElastic(0.2), float32(1), "Elastic springs past the end")
	assert.InDelta(t, 1, ease.OutBounce(1/2.75), 1e-5, "Bounce touches the end")
	assert.InDelta(t, 0.75, ease.OutBounce(1.5/2.75), 1e-5, "then bounces back up a quarter")
	ins := map[string]ease.Func{"Quad": ease.InQuad, "Cubic": ease.InCubic, "Expo": ease.InExpo, "Bounce": ease.InBounce}
	for name, f := range ins {
		assert.Less(t, f(0.3), float32(0.3), "In%s starts slow", name)
	}
}

func run(w *world.World, seconds float32) {
	for range int(seconds * 60) {
		w.Tick(1.0 / 60)
	}
}

func TestATweenDrivesAValueAndFinishes(t *testing.T) {
	w := world.New().AddPlugins(ease.Plugin{})
	var got []float32
	tw := ease.NewTween(1, func(v float32) { got = append(got, v) }, ease.Easing(ease.InQuad), ease.Delay(0.5))
	w.AddEntities(tw)
	var finished world.Reader[ease.TweenFinished]
	var events []ease.TweenFinished
	read := func() {
		for ev := range finished.Read(w) {
			events = append(events, ev)
		}
	}
	run(w, 0.45)
	assert.Empty(t, got, "nothing during the delay")
	run(w, 0.55)
	require.NotEmpty(t, got)
	assert.InDelta(t, 0.25, got[len(got)-1], 0.02, "half way, eased by InQuad")
	assert.InDelta(t, 0.5, tw.Progress(), 0.02, "the progress itself is not eased")
	run(w, 0.6)
	read()
	assert.Equal(t, float32(1), got[len(got)-1], "it ends on 1")
	assert.False(t, tw.Playing())
	require.Len(t, events, 1)
	assert.Equal(t, tw, events[0].Tween)

	_, inWorld := w.Entity(tw.ID())
	assert.False(t, inWorld, "a finished one-shot tween leaves the World")

	tw.Restart()
	w.AddEntities(tw)
	run(w, 0.6)
	assert.InDelta(t, 0.1, tw.Progress(), 0.02, "from the start again, after its delay")
}

func TestLoopAndPingPongRepeatWithoutFinishing(t *testing.T) {
	w := world.New().AddPlugins(ease.Plugin{})
	loop := ease.NewTween(1, nil, ease.Loop())
	pong := ease.NewTween(1, nil, ease.PingPong())
	w.AddEntities(loop, pong)
	var finished world.Reader[ease.TweenFinished]
	run(w, 1.25)
	assert.InDelta(t, 0.25, loop.Progress(), 0.02, "a loop starts over")
	assert.InDelta(t, 0.75, pong.Progress(), 0.02, "a ping-pong comes back")
	run(w, 1)
	assert.InDelta(t, 0.25, pong.Progress(), 0.02)
	assert.True(t, loop.Playing() && pong.Playing())
	n := 0
	for range finished.Read(w) {
		n++
	}
	assert.Zero(t, n)
}

func TestPauseAndResume(t *testing.T) {
	w := world.New().AddPlugins(ease.Plugin{})
	tw := ease.NewTween(1, nil)
	w.AddEntities(tw)
	run(w, 0.25)
	tw.Pause()
	run(w, 1)
	assert.InDelta(t, 0.25, tw.Progress(), 0.02, "paused, it holds")
	tw.Resume()
	run(w, 0.25)
	assert.InDelta(t, 0.5, tw.Progress(), 0.02)
	tw.SetEasing(nil)
	assert.NotNil(t, tw.Easing(), "a nil easing keeps the one it had")
}

func TestOnceUndoesARepeat(t *testing.T) {
	w := world.New().AddPlugins(ease.Plugin{})
	tw := ease.NewTween(0.5, nil, ease.PingPong(), ease.Once())
	w.AddEntities(tw)
	run(w, 1)
	assert.False(t, tw.Playing(), "it ran once and stopped")
}

func TestAFinishedTweenLeavesWhatItMovedWhichStays(t *testing.T) {
	w := world.New().AddPlugins(ease.Plugin{})
	crate := world.NewBasic()
	fade := ease.NewTween(0.5, nil)
	spin := ease.NewTween(0.5, nil, ease.Loop())
	crate.AppendChild(fade)
	crate.AppendChild(spin)
	w.AddEntities(crate)
	run(w, 1)
	_, crateIn := w.Entity(crate.ID())
	_, fadeIn := w.Entity(fade.ID())
	assert.True(t, crateIn, "what it moved stays, for the game to remove on TweenFinished")
	assert.False(t, fadeIn)
	assert.Equal(t, []world.BasicFace{spin}, crate.Children(), "and it is no longer its child; the looping one is")
}
