package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/twod/light"
	"github.com/merlindorin/ember/world"
)

func TestTheSunCrossesTheSkyByDayAndSetsAtNight(t *testing.T) {
	sun := newSun()
	camera := twod.NewCamera(light.Ambient(moonlight()))
	day := &Day{sun: sun, camera: camera}

	day.Update(world.Frame{DT: dayLength / 2})
	if sun.Hidden() || sun.Intensity() < 0.99 || light.AmbientOf(camera) != daylight() {
		t.Fatalf(
			"at noon the sun is full and it is day: hidden %v, intensity %v, ambient %v",
			sun.Hidden(),
			sun.Intensity(),
			light.AmbientOf(camera),
		)
	}
	noon := sun.ShadowLength()
	day.Update(world.Frame{DT: dayLength / 3})
	if sun.ShadowLength() <= noon {
		t.Fatalf("shadows lengthen in the afternoon: %v, then %v", noon, sun.ShadowLength())
	}
	day.Update(world.Frame{DT: dayLength / 3})
	if !sun.Hidden() || light.AmbientOf(camera) != moonlight() {
		t.Fatal("after the day comes the night")
	}
	day.Update(world.Frame{DT: nightLength})
	if sun.Hidden() {
		t.Fatal("and after the night, a new day")
	}
}

func TestSpaceStopsTheClock(t *testing.T) {
	win := platform.NewWindow()
	sun, camera := newSun(), twod.NewCamera(light.Ambient(moonlight()))
	win.Add(sun, camera)
	day := &Day{sun: sun, camera: camera, in: win.Input()}
	w := ember.New().AddEntities(win).AddSystems(day)
	win.Input().SetKey(input.KeySpace, true)
	w.RunFrames(2)
	stopped := day.t
	win.Input().SetKey(input.KeySpace, false)
	w.RunFrames(30)
	if day.t != stopped {
		t.Fatalf("the clock stopped at %v, then ran to %v", stopped, day.t)
	}
}
