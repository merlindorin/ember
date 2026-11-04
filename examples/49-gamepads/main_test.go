package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

func TestPadsWalkTheHeroAndSayWhenTheyArePlugged(t *testing.T) {
	hero, err := newHero()
	if err != nil {
		t.Fatal(err)
	}
	status := twod.NewText("")
	win := platform.NewWindow(platform.Size(760, 460))
	win.Add(newPadView(0), newPadView(1), hero, status)
	in := win.Input()
	w := ember.New().
		AddEntities(win).
		AddSystems(&Walk{hero: hero, in: in, controls: newControls()}, &Plugs{status: status})

	in.ConnectGamepad(0, "Wireless Controller")
	w.RunFrames(2)
	if status.Text() != "Pad 0 plugged in: Wireless Controller" {
		t.Fatalf("status %q", status.Text())
	}

	start := hero.Position()
	in.SetGamepadStick(0, input.LeftStick, geom.V2(0.6, 0))
	w.RunFrames(60)
	in.SetGamepadStick(0, input.LeftStick, geom.Vec2{})
	if d := hero.Position().X - start.X; d < 78 || d > 82 {
		t.Fatalf("a stick leant 0.6, half way past the 0.2 deadzone, walks at half speed: %v", d)
	}

	start = hero.Position()
	in.SetGamepadButton(0, input.GamepadUp, true)
	w.RunFrames(30)
	in.SetGamepadButton(0, input.GamepadUp, false)
	if d := start.Y - hero.Position().Y; d < 78 || d > 82 {
		t.Fatalf("the cross walks at full speed: %v", d)
	}

	in.ConnectGamepad(1, "Pro Controller")
	w.RunFrames(2)
	if len(in.Gamepads()) != 2 || status.Text() != "Pad 1 plugged in: Pro Controller" {
		t.Fatalf("a second pad: %d pads, status %q", len(in.Gamepads()), status.Text())
	}
	in.DisconnectGamepad(0)
	w.RunFrames(2)
	if status.Text() != "Pad 0 unplugged" {
		t.Fatalf("status %q", status.Text())
	}
}

func TestSouthRumblesThePadThatPressedIt(t *testing.T) {
	win := platform.NewWindow(platform.Size(760, 460))
	in := win.Input()
	rec := input.NewRumbleRecorder(in)
	w := ember.New().AddEntities(win).AddSystems(&Rumble{in: in})
	in.ConnectGamepad(0, "Wireless Controller")
	in.ConnectGamepad(1, "Xbox Controller")
	w.RunFrames(2)
	in.SetGamepadButton(1, input.GamepadSouth, true)
	w.RunFrames(2)
	if r, ok := rec.Last(1); !ok || r != (input.Rumble{Low: 1, High: 0.4, Seconds: 0.3}) {
		t.Fatalf("pad 1 rumbles: %+v, %v", r, ok)
	}
	if _, ok := rec.Last(0); ok {
		t.Fatal("pad 0 stays still")
	}
}

func TestTheTriggersDriveTheMotorsLive(t *testing.T) {
	win := platform.NewWindow(platform.Size(760, 460))
	in := win.Input()
	rec := input.NewRumbleRecorder(in)
	w := ember.New().AddEntities(win).AddSystems(&Rumble{in: in})
	in.ConnectGamepad(0, "Wireless Controller")
	w.RunFrames(2)
	in.SetGamepadTrigger(0, input.GamepadLeftTrigger, 0.5)
	w.RunFrames(2)
	if r, _ := rec.Last(0); r.Low != 0.5 || r.High != 0 || r.Seconds <= 0 {
		t.Fatalf("the left trigger drives the heavy motor alone: %+v", r)
	}
	in.SetGamepadTrigger(0, input.GamepadLeftTrigger, 0)
	in.SetGamepadTrigger(0, input.GamepadRightTrigger, 1)
	w.RunFrames(2)
	if r, _ := rec.Last(0); r.Low != 0 || r.High != 1 {
		t.Fatalf("the right trigger drives the light motor alone: %+v", r)
	}
	in.SetGamepadTrigger(0, input.GamepadRightTrigger, 0)
	w.RunFrames(2)
	if r, ok := rec.Last(0); !ok || r != (input.Rumble{}) {
		t.Fatalf("letting go stops the pad: %+v", r)
	}
}
