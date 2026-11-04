package input //nolint:testpackage // white-box: the unexported feed, the backend side of a State

import (
	"testing"

	"github.com/merlindorin/ember/geom"
)

func TestPressedOnlyOnTransitionFrame(t *testing.T) {
	var s State
	s.SetKey(KeySpace, true)
	fed(&s).BeginFrame()
	if !s.Pressed(KeySpace) || !s.Down(KeySpace) {
		t.Fatal("first frame must report pressed")
	}
	fed(&s).BeginFrame()
	if s.Pressed(KeySpace) || !s.Down(KeySpace) {
		t.Fatal("held key must not report pressed again")
	}
	s.SetKey(KeySpace, false)
	fed(&s).BeginFrame()
	if !s.Released(KeySpace) {
		t.Fatal("release not detected")
	}
}

func TestInvalidKeysAreIgnored(t *testing.T) {
	var s State
	s.SetKey(KeyUnknown, true)
	s.SetKey(Key(9999), true)
	if s.Down(KeyUnknown) || s.Pressed(Key(9999)) {
		t.Fatal("invalid keys must read as up")
	}
}

func TestVectorIsNormalized(t *testing.T) {
	var s State
	s.SetKey(KeyD, true)
	s.SetKey(KeyW, true)
	fed(&s).BeginFrame()
	v := s.Direction(KeyA, KeyD, KeyW, KeyS)
	if geom.Abs(v.Len()-1) > 1e-5 || v.X <= 0 || v.Y >= 0 {
		t.Fatalf("got %v", v)
	}
}

func TestScrollAndMouseDelta(t *testing.T) {
	var s State
	s.SetMouse(geom.V2(10, 10))
	fed(&s).BeginFrame()
	s.SetMouse(geom.V2(15, 7))
	s.AddWheel(geom.V2(0, 1))
	s.AddWheel(geom.V2(0, 2))
	fed(&s).BeginFrame()
	if s.MouseDelta() != geom.V2(5, -3) || s.Wheel() != geom.V2(0, 3) {
		t.Fatalf("delta %v scroll %v", s.MouseDelta(), s.Wheel())
	}
	fed(&s).BeginFrame()
	if s.Wheel() != (geom.Vec2{}) {
		t.Fatal("scroll must reset each frame")
	}
}

func TestKeyboardCaptureHidesKeysFromGameplay(t *testing.T) {
	var s State
	s.SetKey(KeyW, true)
	fed(&s).BeginFrame()
	s.keyboardCaptured = true
	if s.Down(KeyW) || s.Pressed(KeyW) {
		t.Fatal("captured keyboard must hide keys from gameplay")
	}
	if !fed(&s).RawDown(int(KeyW)) || !fed(&s).RawPressed(int(KeyW)) {
		t.Fatal("raw accessors must still see keys")
	}
	fed(&s).BeginFrame()
	if s.KeyboardCaptured() || !s.Down(KeyW) {
		t.Fatal("capture resets every frame")
	}
}

func TestKeyNames(t *testing.T) {
	for _, c := range []struct {
		k    Key
		want string
	}{
		{KeyF1, "F1"}, {KeyF12, "F12"}, {KeyA, "A"}, {Key7, "7"},
		{KeyEscape, "Escape"}, {KeyPad3, "Pad3"}, {Key(999), "Key(999)"},
	} {
		if got := c.k.String(); got != c.want {
			t.Fatalf("%d: %q want %q", int(c.k), got, c.want)
		}
	}
}

func TestDropsLastOneFrame(t *testing.T) {
	var s State
	s.AddDrop("/tmp/a.glb", "/tmp/pack.zip")
	if len(s.Dropped()) != 0 {
		t.Fatal("drops are published at the next frame")
	}
	fed(&s).BeginFrame()
	if got := s.Dropped(); len(got) != 2 || got[1] != "/tmp/pack.zip" {
		t.Fatalf("dropped %v", got)
	}
	fed(&s).BeginFrame()
	if len(s.Dropped()) != 0 {
		t.Fatal("a drop is seen once")
	}
}

func TestRepeatsLastOneFrame(t *testing.T) {
	var s State
	fed(&s).AddRepeat(int(KeyBackspace))
	if fed(&s).RawRepeated(int(KeyBackspace)) {
		t.Fatal("a repeat shows from the next frame")
	}
	fed(&s).BeginFrame()
	if !fed(&s).RawRepeated(int(KeyBackspace)) || fed(&s).RawRepeated(int(KeyLeft)) {
		t.Fatal("the repeated key only")
	}
	fed(&s).BeginFrame()
	if fed(&s).RawRepeated(int(KeyBackspace)) {
		t.Fatal("repeats last one frame")
	}
}

func fed(s *State) feed { return feed{s} }
