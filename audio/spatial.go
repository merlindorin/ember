package audio

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/scene"
	"github.com/lantervale/ember/world"
)

// A Listener hears emitters from where its entity is, turned as it is: put
// it on the camera, or on the hero. The first one in the World is used. In
// 3D it faces its transform's -Z; in 2D it faces up (-Y) at rotation 0, and
// its placement's rotation turns it, clockwise as the angle grows, so a
// sound on its right moves to its left after a half turn.
//
//	type Eye struct {
//		*world.BasicEntity
//		scene.Transform
//		render.Camera
//		audio.Listener
//	}
type Listener struct{}

func (l *Listener) GetSoundListener() *Listener { return l }

// An Emitter makes its entity sound where it is, louder as the listener
// comes near and from the listener's side. The entity places it with a
// scene.Transform in 3D or a 2D placement (twod entities).
//
//	type Campfire struct {
//		twod.Item
//		*audio.Emitter
//	}
type Emitter struct {
	sound   Sound
	set     settings
	voice   *Voice
	want    bool
	started bool
	gain    float32
	pan     float32
}

// NewEmitter plays sound with the voice options and Distance, Rolloff and
// Autoplay.
func NewEmitter(sound Sound, opts ...Option) *Emitter {
	return &Emitter{sound: sound, set: newSettings(opts), gain: 1}
}

func (e *Emitter) GetSoundEmitter() *Emitter { return e }

func (e *Emitter) Sound() Sound { return e.sound }

// Play starts the emitter again from the beginning, on the next frame.
func (e *Emitter) Play() { e.want = true }

// Stop ends its sound, over FadeOut when given.
func (e *Emitter) Stop(opts ...Option) {
	e.want = false
	if e.voice != nil {
		e.voice.Stop(opts...)
	}
}

func (e *Emitter) Playing() bool { return e.want || e.voice != nil && e.voice.Playing() }

// Voice is the emitter's current voice, or nil before it first plays.
func (e *Emitter) Voice() *Voice { return e.voice }

// Distance is where the emitter starts fading and where it falls silent.
func (e *Emitter) Distance() (float32, float32) { return e.set.minDistance, e.set.maxDistance }

func (e *Emitter) SetDistance(minDist, maxDist float32) {
	e.set.minDistance, e.set.maxDistance = clampDistance(minDist, maxDist)
}

func (e *Emitter) Rolloff() float32     { return e.set.rolloff }
func (e *Emitter) SetRolloff(r float32) { e.set.rolloff = max(r, 0) }

func (e *Emitter) Autoplay() bool { return e.set.autoplay }

// Heard is the gain and pan the listener heard last frame: 1 and the Pan
// option with no listener.
func (e *Emitter) Heard() (float32, float32) { return e.gain, e.pan }

func (e *Emitter) Volume() float32   { return e.set.volume }
func (e *Emitter) Pan() float32      { return e.set.pan }
func (e *Emitter) Pitch() float32    { return e.set.pitch }
func (e *Emitter) Loop() bool        { return e.set.loop }
func (e *Emitter) Bus() string       { return e.set.bus }
func (e *Emitter) Priority() int     { return e.set.priority }
func (e *Emitter) Effects() []Effect { return e.set.effects }

// SetVolume changes the emitter's volume, and its voice's.
func (e *Emitter) SetVolume(v float32) {
	e.set.volume = max(v, 0)
	if e.voice != nil {
		e.voice.SetVolume(e.set.volume)
	}
}

// Attenuation is the gain at distance d: 1 within minDist, then
// minDist/(minDist+rolloff·(d−minDist)), and silence from maxDist.
func Attenuation(d, minDist, maxDist, rolloff float32) float32 {
	switch {
	case d <= minDist:
		return 1
	case d >= maxDist:
		return 0
	}
	return minDist / (minDist + rolloff*(d-minDist))
}

type listening interface {
	world.Identifier
	GetSoundListener() *Listener
}

type emitting interface {
	world.Identifier
	GetSoundEmitter() *Emitter
}

type placed3D interface{ GetTransform() *scene.Transform }

type placed2D interface{ Global() geom.Transform2D }

// SpatialSystem plays the emitters of a World and sets their gain and pan
// from the listener, in PostUpdate once transforms are resolved.
type SpatialSystem struct {
	world.Group[emitting]
	listeners world.Group[listening]
	mixers    *world.Group[*Mixer]
}

func (*SpatialSystem) Phase() world.Phase { return world.PostUpdate }
func (*SpatialSystem) Priority() int      { return -500 }

func (s *SpatialSystem) Matches(o world.Identifier) bool {
	return s.Group.Matches(o) || s.listeners.Matches(o)
}

func (s *SpatialSystem) AddByInterface(o world.Identifier) {
	s.Group.AddByInterface(o)
	s.listeners.AddByInterface(o)
}

func (s *SpatialSystem) Remove(o world.Identifier) {
	if e, ok := s.Get(o.ID()); ok && e.GetSoundEmitter().voice != nil {
		e.GetSoundEmitter().voice.Stop()
	}
	s.Group.Remove(o)
	s.listeners.Remove(o)
}

func (s *SpatialSystem) Update(world.Frame) {
	var mixer *Mixer
	if s.mixers != nil {
		for m := range s.mixers.All() {
			mixer = m
			break
		}
	}
	var ear listening
	for l := range s.listeners.All() {
		ear = l
		break
	}
	for ent := range s.All() {
		e := ent.GetSoundEmitter()
		spatial := ear != nil
		e.gain, e.pan = 1, e.set.pan
		if ear != nil {
			e.gain, e.pan, spatial = hear(ear, ent, e.set)
		}
		if mixer == nil {
			continue
		}
		mixer.mu.Lock()
		if e.want || e.set.autoplay && !e.started {
			if e.voice != nil && e.voice.state < stopping {
				e.voice.stop(mixer.frames(smoothing), false)
			}
			e.voice = mixer.play(e.sound, e.set)
			e.want, e.started = false, true
			if spatial {
				e.voice.setSpatial(e.gain, e.pan)
				e.voice.gl, e.voice.gr = e.voice.target()
			}
		} else if e.voice != nil && spatial {
			e.voice.setSpatial(e.gain, e.pan)
		}
		mixer.mu.Unlock()
	}
}

func hear(ear, ent any, set settings) (float32, float32, bool) {
	at, ok := position(ent)
	if !ok {
		return 1, set.pan, false
	}
	from, toLocal, ok := frame(ear)
	if !ok {
		return 1, set.pan, false
	}
	d := at.Sub(from).Len()
	gain := Attenuation(d, set.minDistance, set.maxDistance, set.rolloff)
	var pan float32
	if local := toLocal(at); local.Len() > 1e-6 {
		pan = clampPan(local.X / local.Len())
	}
	return gain, pan, true
}

func position(e any) (geom.Vec3, bool) {
	switch p := e.(type) {
	case placed3D:
		return p.GetTransform().WorldPosition(), true
	case placed2D:
		o := p.Global().Origin
		return geom.V3(o.X, o.Y, 0), true
	}
	return geom.Vec3{}, false
}

func frame(e any) (geom.Vec3, func(geom.Vec3) geom.Vec3, bool) {
	switch p := e.(type) {
	case placed3D:
		m := p.GetTransform().World()
		inv := m.Inverse()
		return m.Translation(), inv.MulPoint, true
	case placed2D:
		g := p.Global()
		inv := g.Inverse()
		return geom.V3(g.Origin.X, g.Origin.Y, 0), func(v geom.Vec3) geom.Vec3 {
			l := inv.Xform(geom.V2(v.X, v.Y))
			return geom.V3(l.X, l.Y, 0)
		}, true
	}
	return geom.Vec3{}, nil, false
}
