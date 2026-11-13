package main

import (
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
)

func TestSpaceAndClicksPlayTheSound(t *testing.T) {
	hit, err := audio.Load(files, "hit.wav")
	if err != nil {
		t.Fatal(err)
	}
	if hit.Duration().Seconds() < 0.3 {
		t.Fatalf("the pling lasts %v", hit.Duration())
	}
	mixer := audio.NewMixer()
	rec := audio.NewRecorder(mixer)
	win := platform.NewWindow(platform.Size(640, 360))
	player := newPlayer(win, mixer, hit)
	win.Add(player.layer)
	w := ember.New().AddEntities(win, mixer).AddSystems(player)
	in := win.Input()

	in.SetKey(input.KeySpace, true)
	w.RunFrames(1)
	in.SetKey(input.KeySpace, false)
	w.RunFrames(5)
	if l, r := rec.PeakLR(0, rec.Frames()); l < 0.3 || l != r || mixer.Voices() != 1 {
		t.Fatalf("Space plays the pling in the middle: %v %v, %d voices", l, r, mixer.Voices())
	}

	w.RunFrames(30)
	from := rec.Frames()
	in.SetMouse(geom.V2(600, 100))
	in.SetMouseButton(input.MouseLeft, true)
	w.RunFrames(1)
	in.SetMouseButton(input.MouseLeft, false)
	w.RunFrames(5)
	if l, r := rec.PeakLR(from, rec.Frames()); r < 0.3 || l > r/4 {
		t.Fatalf("a click near the right edge plays on the right: %v %v", l, r)
	}

	w.RunFrames(30)
	if player.played != 2 || mixer.Voices() != 0 {
		t.Fatalf("both plings finish: %d finished, %d playing", player.played, mixer.Voices())
	}
	if !player.rings[0].line.Hidden() || player.status.Text() == "" {
		t.Fatal("the rings fade out and the status shows the counts")
	}
}
