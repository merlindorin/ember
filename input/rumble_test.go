package input_test

import (
	"testing"

	"github.com/merlindorin/ember/internal/inputfeed"
	"github.com/stretchr/testify/assert"

	"github.com/merlindorin/ember/input"
)

func padOn(s *input.State, id int) *input.Gamepad {
	s.ConnectGamepad(id, "DUALSHOCK 4 Wireless Controller")
	inputfeed.Of(s).BeginFrame()
	return s.Gamepad(id)
}

func TestARumbleIsRecordedAndSentToTheBackend(t *testing.T) {
	var s input.State
	rec := input.NewRumbleRecorder(&s)
	pad := padOn(&s, 1)
	_, played := rec.Last(1)
	assert.False(t, played)

	pad.Rumble(0.8, 0.3, 0.25)
	got, played := rec.Last(1)
	assert.True(t, played)
	assert.Equal(t, input.Rumble{Low: 0.8, High: 0.3, Seconds: 0.25}, got)
	assert.Equal(t, []inputfeed.Rumble{{Pad: 1, Low: 0.8, High: 0.3, Seconds: 0.25}}, inputfeed.Of(&s).Rumbles())
	assert.Empty(t, inputfeed.Of(&s).Rumbles(), "the backend takes each rumble once")
}

func TestANewRumbleReplacesTheLastOneOfThatPad(t *testing.T) {
	var s input.State
	rec := input.NewRumbleRecorder(&s)
	padOn(&s, 0)
	pad := padOn(&s, 2)
	s.Gamepad(0).Rumble(1, 1, 1)
	pad.Rumble(1, 0, 1)
	pad.Rumble(0, 0.5, 2)
	assert.Equal(t,
		[]inputfeed.Rumble{{Pad: 0, Low: 1, High: 1, Seconds: 1}, {Pad: 2, High: 0.5, Seconds: 2}},
		inputfeed.Of(&s).Rumbles())
	got, _ := rec.Last(2)
	assert.Equal(t, input.Rumble{High: 0.5, Seconds: 2}, got)
}

func TestStopRumbleSendsAStop(t *testing.T) {
	var s input.State
	rec := input.NewRumbleRecorder(&s)
	pad := padOn(&s, 0)
	pad.Rumble(1, 1, 1)
	pad.StopRumble()
	got, played := rec.Last(0)
	assert.True(t, played)
	assert.Equal(t, input.Rumble{}, got)
	assert.Equal(t, []inputfeed.Rumble{{Pad: 0}}, inputfeed.Of(&s).Rumbles())
}

func TestRumbleIsClampedAndSilentOnesStop(t *testing.T) {
	var s input.State
	rec := input.NewRumbleRecorder(&s)
	pad := padOn(&s, 0)
	pad.Rumble(2, -1, 99)
	got, _ := rec.Last(0)
	assert.Equal(t, input.Rumble{Low: 1, Seconds: input.MaxRumbleSeconds}, got)
	pad.Rumble(0, 0, 1)
	got, _ = rec.Last(0)
	assert.Equal(t, input.Rumble{}, got, "no strength is a stop")
	pad.Rumble(1, 1, 0)
	got, _ = rec.Last(0)
	assert.Equal(t, input.Rumble{}, got, "no time is a stop")
}

func TestRumbleOffMakesEveryCallANoOp(t *testing.T) {
	var s input.State
	assert.True(t, s.RumbleEnabled())
	rec := input.NewRumbleRecorder(&s)
	pad := padOn(&s, 3)
	pad.Rumble(1, 1, 5)
	inputfeed.Of(&s).Rumbles()

	s.SetRumbleEnabled(false)
	assert.False(t, s.RumbleEnabled())
	assert.Equal(t, []inputfeed.Rumble{{Pad: 3}}, inputfeed.Of(&s).Rumbles(), "turning it off stops the pads")
	rec.Reset()
	pad.Rumble(1, 1, 1)
	pad.StopRumble()
	_, played := rec.Last(3)
	assert.False(t, played)
	assert.Empty(t, inputfeed.Of(&s).Rumbles())

	s.SetRumbleEnabled(true)
	pad.Rumble(0.5, 0.5, 1)
	_, played = rec.Last(3)
	assert.True(t, played)
}

func TestOnlyAConnectedPadRumbles(t *testing.T) {
	var s input.State
	rec := input.NewRumbleRecorder(&s)
	s.Gamepad(4).Rumble(1, 1, 1)
	s.Gamepad(99).Rumble(1, 1, 1)
	_, played := rec.Last(4)
	assert.False(t, played)
	assert.Empty(t, inputfeed.Of(&s).Rumbles())
	_, played = rec.Last(99)
	assert.False(t, played)
}
