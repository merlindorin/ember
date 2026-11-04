package audio

import "time"

// An Option sets how a sound plays: for Mixer.Play, Mixer.PlayMusic,
// Voice.Stop and NewEmitter. Each says where it applies; elsewhere it is
// ignored.
type Option func(*settings)

type settings struct {
	volume, pan, pitch float32
	loop               bool
	bus                string
	fadeIn, fadeOut    time.Duration
	crossfade          time.Duration
	priority           int
	effects            []Effect

	minDistance, maxDistance, rolloff float32
	autoplay                          bool
}

func newSettings(opts []Option) settings {
	s := settings{volume: 1, pitch: 1, minDistance: 1, maxDistance: 100, rolloff: 1}
	for _, o := range opts {
		o(&s)
	}
	return s
}

// Volume scales the sound, 1 by default.
func Volume(v float32) Option { return func(s *settings) { s.volume = max(v, 0) } }

// Pan places the sound from -1 (left) to 1 (right); 0, the default, leaves
// it as it is. Emitters pan from their position instead.
func Pan(p float32) Option { return func(s *settings) { s.pan = clampPan(p) } }

// Pitch plays the sound faster and higher (2 is an octave up), or slower and
// lower; 1 by default.
func Pitch(p float32) Option { return func(s *settings) { s.pitch = clampPitch(p) } }

// Loop plays the sound again each time it ends, until it is stopped.
func Loop() Option { return func(s *settings) { s.loop = true } }

// OnBus plays the sound on the named bus; unknown names play on master.
func OnBus(name string) Option { return func(s *settings) { s.bus = name } }

// FadeIn raises the sound from silence over d when it starts.
func FadeIn(d time.Duration) Option { return func(s *settings) { s.fadeIn = max(d, 0) } }

// FadeOut lowers the sound to silence over d, for Voice.Stop and
// Mixer.StopMusic.
func FadeOut(d time.Duration) Option { return func(s *settings) { s.fadeOut = max(d, 0) } }

// Crossfade blends the new track with the one playing over d, at equal
// power, for Mixer.PlayMusic.
func Crossfade(d time.Duration) Option { return func(s *settings) { s.crossfade = max(d, 0) } }

// Priority keeps the sound over lower ones when the mixer runs out of
// voices; 0 by default.
func Priority(n int) Option { return func(s *settings) { s.priority = n } }

// Distance is where an emitter starts fading (minDist) and where it falls
// silent (maxDist), 1 and 100 by default, in the world's units: metres in
// 3D, points in 2D.
func Distance(minDist, maxDist float32) Option {
	return func(s *settings) { s.minDistance, s.maxDistance = clampDistance(minDist, maxDist) }
}

// Rolloff is how fast an emitter fades past its min distance, 1 by default.
func Rolloff(r float32) Option { return func(s *settings) { s.rolloff = max(r, 0) } }

// Autoplay starts an emitter as soon as it is in a World with a Mixer.
func Autoplay() Option { return func(s *settings) { s.autoplay = true } }

func clampPan(p float32) float32 { return min(max(p, -1), 1) }

func clampPitch(p float32) float32 { return min(max(p, 1.0/16), 16) }

func clampDistance(lo, hi float32) (float32, float32) {
	lo = max(lo, 1e-3)
	return lo, max(hi, lo)
}
