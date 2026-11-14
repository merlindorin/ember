package audio_test

import (
	"math"
	"testing"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

type ear struct {
	*world.BasicEntity
	scene.Transform
	audio.Listener
}

type source3D struct {
	*world.BasicEntity
	scene.Transform
	*audio.Emitter
}

type flat struct {
	*world.BasicEntity
	at   geom.Vec2
	turn float32
}

func (f *flat) Global() geom.Transform2D {
	return geom.Transform2DFrom(f.at, f.turn, 0, geom.V2(1, 1))
}

type ear2D struct {
	flat
	audio.Listener
}

type source2D struct {
	flat
	*audio.Emitter
}

func TestAttenuationIsClampedInverseDistance(t *testing.T) {
	for _, c := range []struct{ d, want float32 }{{0, 1}, {2, 1}, {4, 0.5}, {10, 0.2}, {50, 0}, {80, 0}} {
		if got := audio.Attenuation(c.d, 2, 50, 1); !near(got, c.want, 1e-6) {
			t.Fatalf("at %v: %v, want %v", c.d, got, c.want)
		}
	}
	if got := audio.Attenuation(4, 2, 50, 2); !near(got, 1.0/3, 1e-6) {
		t.Fatalf("rolloff 2 fades faster: %v", got)
	}
}

func TestEmittersAreHeardFromTheListenersSide(t *testing.T) {
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	c := mono(t, rate, rate, 1)
	listener := &ear{BasicEntity: world.NewBasic(), Transform: scene.At(0, 0, 0)}
	right := &source3D{
		world.NewBasic(),
		scene.At(4, 0, 0),
		audio.NewEmitter(c, audio.Autoplay(), audio.Loop(), audio.Distance(1, 20)),
	}
	left := &source3D{world.NewBasic(), scene.At(-2, 0, 0), audio.NewEmitter(c, audio.Distance(1, 20))}
	behind := &source3D{world.NewBasic(), scene.At(0, 0, 3), audio.NewEmitter(c, audio.Distance(1, 20))}
	far := &source3D{world.NewBasic(), scene.At(0, 0, -30), audio.NewEmitter(c, audio.Autoplay(), audio.Distance(1, 20))}
	w := world.New().AddPlugins(audio.Plugin{}).AddEntities(m, listener, right, left, behind, far)
	w.RunFrames(1)

	if !right.Playing() || left.Playing() || !far.Playing() {
		t.Fatal("autoplay emitters start, the others wait for Play")
	}
	if !near(gain(right), 0.25, 1e-6) || !near(pan(right), 1, 1e-6) {
		t.Fatalf("4 m to the right: gain %v pan %v", gain(right), pan(right))
	}
	if !near(gain(left), 0.5, 1e-6) || !near(pan(left), -1, 1e-6) {
		t.Fatalf("2 m to the left: gain %v pan %v", gain(left), pan(left))
	}
	if !near(pan(behind), 0, 1e-6) || gain(far) != 0 {
		t.Fatalf("behind is centred (%v), beyond max is silent (%v)", pan(behind), gain(far))
	}
	w.RunFrames(2)
	l, r := rec.PeakLR(rec.Frames()-100, rec.Frames())
	if !near(l, 0, 1e-6) || !near(r, 0.25, 1e-6) {
		t.Fatalf("only the right emitter is heard, in the right ear: %v %v", l, r)
	}

	listener.Rotation = geom.QuatYaw(math.Pi)
	w.RunFrames(3)
	if !near(pan(right), -1, 1e-5) {
		t.Fatalf("turning around puts the right emitter on the left: %v", pan(right))
	}
	l, r = rec.PeakLR(rec.Frames()-100, rec.Frames())
	if !near(l, 0.25, 1e-6) || !near(r, 0, 1e-6) {
		t.Fatalf("the mix follows the listener: %v %v", l, r)
	}

	left.Play()
	w.RunFrames(1)
	if !left.Playing() || left.Voice() == nil {
		t.Fatal("Play starts an emitter")
	}
	voice := right.Voice()
	w.RemoveEntity(right)
	w.RunFrames(2)
	if voice.Playing() {
		t.Fatal("removing an emitter stops its sound")
	}
}

func TestEmittersIn2D(t *testing.T) {
	m := audio.NewMixer()
	c := mono(t, rate, rate, 1)
	hero := &ear2D{flat: flat{BasicEntity: world.NewBasic(), at: geom.V2(100, 100)}}
	fire := &source2D{
		flat{BasicEntity: world.NewBasic(), at: geom.V2(40, 100)},
		audio.NewEmitter(c, audio.Autoplay(), audio.Distance(30, 300)),
	}
	w := world.New().AddPlugins(audio.Plugin{}).AddEntities(m, hero, fire)
	w.RunFrames(1)
	if !near(gain(fire), 0.5, 1e-6) || !near(pan(fire), -1, 1e-6) {
		t.Fatalf("60 points to the left: gain %v pan %v", gain(fire), pan(fire))
	}
	hero.at = geom.V2(40, 160)
	w.RunFrames(1)
	if !near(gain(fire), 0.5, 1e-6) || !near(pan(fire), 0, 1e-6) {
		t.Fatalf("straight above is centred: gain %v pan %v", gain(fire), pan(fire))
	}
	hero.at = geom.V2(100, 100)
	for _, c := range []struct{ turn, pan float32 }{{math.Pi / 2, 0}, {math.Pi, 1}, {-math.Pi / 2, 0}} {
		hero.turn = c.turn
		w.RunFrames(1)
		if !near(pan(fire), c.pan, 1e-5) {
			t.Fatalf("turned %v: the fire on the left pans %v, want %v", c.turn, pan(fire), c.pan)
		}
	}
	hero.turn = math.Pi / 2
	fire.at = geom.V2(100, 160)
	w.RunFrames(1)
	if !near(pan(fire), 1, 1e-5) {
		t.Fatalf("facing right, a sound below is on the right ear: %v", pan(fire))
	}

	fire.SetDistance(60, 300)
	fire.SetRolloff(2)
	if a, b := fire.Distance(); a != 60 || b != 300 || fire.Rolloff() != 2 || !fire.Autoplay() {
		t.Fatal("emitter settings read back")
	}
	fire.SetVolume(0.5)
	if fire.Volume() != 0.5 || fire.Voice().Volume() != 0.5 || fire.Loop() || fire.Bus() != "" ||
		fire.Pitch() != 1 || fire.Priority() != 0 || fire.Pan() != 0 {
		t.Fatal("SetVolume reaches the voice, and the voice options read back")
	}
}

func TestWithoutAListenerEmittersPlayAsSet(t *testing.T) {
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	e := &source3D{
		world.NewBasic(),
		scene.At(50, 0, 0),
		audio.NewEmitter(mono(t, rate, rate, 0.5), audio.Autoplay(), audio.Pan(-1)),
	}
	w := world.New().AddPlugins(audio.Plugin{}).AddEntities(m, e)
	w.RunFrames(2)
	if l, r := rec.PeakLR(0, rec.Frames()); !near(l, 0.5, 1e-6) || r != 0 || gain(e) != 1 {
		t.Fatalf("no listener: plain voice %v %v", l, r)
	}
	e.Stop()
	w.RunFrames(2)
	if e.Playing() {
		t.Fatal("Stop ends an emitter")
	}
}

type hearer interface{ Heard() (gain, pan float32) }

func gain(e hearer) float32 { g, _ := e.Heard(); return g }
func pan(e hearer) float32  { _, p := e.Heard(); return p }
