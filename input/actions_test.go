package input_test

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/inputfeed"
)

type act int

const (
	jump act = iota
	left
	right
	fire
)

func (a act) String() string { return [...]string{"jump", "left", "right", "fire"}[a] }

func frame(s *input.State, set func()) {
	set()
	inputfeed.Of(s).BeginFrame()
}

func TestOneTrioReadsKeysAndMouseButtons(t *testing.T) {
	var s input.State
	frame(&s, func() { s.SetKey(input.KeyRightShift, true); s.SetMouseButton(input.MouseBack, true) })
	assert.True(t, s.Down(input.KeyShift), "either side's shift")
	assert.True(t, s.Pressed(input.KeyShift))
	assert.True(t, s.Pressed(input.MouseBack))
	frame(&s, func() { s.SetMouseButton(input.MouseBack, false) })
	assert.True(t, s.Released(input.MouseBack))
	assert.True(t, s.Down(input.KeyShift) && !s.Pressed(input.KeyShift), "still held, not pressed again")
	inputfeed.Of(&s).CaptureKeyboard()
	assert.False(t, s.Down(input.KeyShift), "a text field hides keys")
	assert.True(t, inputfeed.Of(&s).RawDown(int(input.KeyShift)), "but not from ui")
}

func TestButtonsHaveNames(t *testing.T) {
	for name, b := range map[string]input.Button{
		"Space": input.KeySpace, "W": input.KeyW, "Pad3": input.KeyPad3, "Shift": input.KeyShift,
		"MouseLeft": input.MouseLeft, "MouseBack": input.MouseBack, "F5": input.KeyF5,
	} {
		got, err := input.ParseButton(name)
		require.NoError(t, err, name)
		assert.Equal(t, b, got, name)
		assert.Equal(t, name, b.String())
	}
	_, err := input.ParseButton("Jump")
	assert.Error(t, err)
}

func TestActionsReadIntentAndRebind(t *testing.T) {
	controls := input.NewActions(
		input.Bind(jump, input.KeySpace, input.KeyW),
		input.Bind(left, input.KeyA, input.KeyLeft),
		input.Bind(right, input.KeyD, input.KeyRight),
		input.Bind(fire, input.MouseLeft),
	)
	var s input.State
	frame(&s, func() {
		s.SetKey(input.KeyW, true)
		s.SetKey(input.KeyLeft, true)
		s.SetMouseButton(input.MouseLeft, true)
	})
	assert.True(t, controls.Pressed(&s, jump), "W jumps too")
	assert.True(t, controls.Down(&s, fire))
	assert.Equal(t, float32(-1), controls.Axis(&s, left, right))
	assert.Equal(t, geom.V2(-1, 0), controls.Direction(&s, left, right, jump, jump))

	frame(&s, func() { s.SetKey(input.KeySpace, true) })
	assert.False(t, controls.Pressed(&s, jump), "another of its keys does not press it again")
	frame(&s, func() { s.SetKey(input.KeySpace, false); s.SetKey(input.KeyW, false) })
	assert.True(t, controls.Released(&s, jump))

	controls.Bind(jump, input.KeyUp)
	assert.Equal(t, []input.Button{input.KeyUp}, controls.Bindings(jump), "a rebinding replaces")
	bound, ok := controls.Bound(input.MouseLeft)
	assert.True(t, ok)
	assert.Equal(t, fire, bound)
	_, ok = controls.Bound(input.KeyW)
	assert.False(t, ok)
	assert.Equal(t, []act{jump, left, right, fire}, controls.Actions())
}

func TestActionsSaveTheirBindingsByName(t *testing.T) {
	controls := input.NewActions(input.Bind(jump, input.KeySpace), input.Bind(fire, input.MouseLeft, input.KeyControl))
	data, err := json.Marshal(controls)
	require.NoError(t, err)
	assert.JSONEq(t, `{"jump":["Space"],"fire":["MouseLeft","Control"]}`, string(data))

	settings := input.NewActions(input.Bind(jump, input.KeyW), input.Bind(fire, input.MouseRight))
	require.NoError(t, json.Unmarshal([]byte(`{"jump":["Space","Up"],"dash":["Shift"]}`), settings))
	assert.Equal(t, []input.Button{input.KeySpace, input.KeyUp}, settings.Bindings(jump), "saved bindings replace")
	assert.Equal(t, []input.Button{input.MouseRight}, settings.Bindings(fire),
		"unsaved ones stay, unknown ones are ignored")
	assert.Error(t, json.Unmarshal([]byte(`{"jump":["Nope"]}`), settings))
}
