package input_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/input"
)

func TestChangedIsWhetherAnythingArrivedThisFrame(t *testing.T) {
	s := &input.State{}
	f := inputfeed.Of(s)
	step := func(act func()) bool {
		if act != nil {
			act()
		}
		f.BeginFrame()
		return f.Changed()
	}
	step(nil)
	assert.False(t, step(nil), "a quiet frame")
	for _, c := range []struct {
		name string
		act  func()
	}{
		{"a key down", func() { s.SetKey(input.KeyA, true) }},
		{"a key up", func() { s.SetKey(input.KeyA, false) }},
		{"a repeat", func() { f.AddRepeat(int(input.KeyB)) }},
		{"a mouse button", func() { s.SetMouseButton(input.MouseLeft, true) }},
		{"its release", func() { s.SetMouseButton(input.MouseLeft, false) }},
		{"the pointer", func() { s.SetMouse(geom.V2(10, 20)) }},
		{"the wheel", func() { s.AddWheel(geom.V2(0, 1)) }},
		{"text", func() { s.AddText('a') }},
	} {
		assert.True(t, step(c.act), c.name)
		assert.False(t, step(nil), "%s, then quiet again", c.name)
	}
	assert.True(t, step(func() { s.ConnectGamepad(0, "pad") }), "a pad connecting")
	step(nil)
	assert.True(t, step(func() { s.SetGamepadButton(0, input.GamepadSouth, true) }), "a pad button")
	step(nil)
	assert.True(t, step(func() { s.SetGamepadStick(0, input.LeftStick, geom.V2(0.5, 0)) }), "a stick")
	assert.False(t, step(nil), "a stick held still")
}
