// Package audio plays sounds: clips and streams through a Mixer, its voices
// and buses, music with crossfades, and emitters heard from a listener.
//
//	hit, _ := audio.Load(files, "hit.wav")   // import _ ".../audio/wav"
//	mixer := audio.NewMixer()
//	ember.New().AddEntities(win, mixer)
//	mixer.Play(hit, audio.Pitch(1.1))
//
// The package is pure Go: servers and tests mix into a null output at the
// World's pace, and a Recorder keeps what was mixed for tests to check.
package audio

import "github.com/lantervale/ember/world"

// Plugin runs the mixers and emitters of a World. ember.New installs it;
// servers add it to world.New.
type Plugin struct{}

func (Plugin) Build(w *world.World) {
	mix := &MixSystem{}
	w.AddSystems(mix, &SpatialSystem{mixers: &mix.Group})
}

// MixSystem advances the mixers that no device reads and emits their events,
// first thing in PreUpdate, so games read VoiceFinished and VoiceLooped in Update.
type MixSystem struct {
	world.Group[*Mixer]
}

func (*MixSystem) Phase() world.Phase { return world.PreUpdate }
func (*MixSystem) Priority() int      { return 1000 }

func (s *MixSystem) Update(f world.Frame) {
	for m := range s.All() {
		m.advance(f.DT)
		for _, ev := range m.drain() {
			switch ev := ev.(type) {
			case VoiceFinished:
				world.Emit(f.World, ev)
			case VoiceLooped:
				world.Emit(f.World, ev)
			}
		}
	}
}

// A Recorder keeps everything a mixer mixes into its null output, for tests:
// run the World's frames, or Mix directly, then check the samples.
type Recorder struct {
	mixer   *Mixer
	samples []float32
}

// NewRecorder records m.
func NewRecorder(m *Mixer) *Recorder {
	r := &Recorder{mixer: m}
	m.mu.Lock()
	m.recorder = r
	m.mu.Unlock()
	return r
}

// Mix mixes the next frames now.
func (r *Recorder) Mix(frames int) { r.mixer.mix(make([]float32, 2*max(frames, 0))) }

// Samples are the stereo samples mixed so far, interleaved.
func (r *Recorder) Samples() []float32 { return r.samples }

func (r *Recorder) Frames() int { return len(r.samples) / 2 }

// Peak is the loudest sample of either channel in frames [from, to).
func (r *Recorder) Peak(from, to int) float32 {
	l, rr := r.PeakLR(from, to)
	return max(l, rr)
}

// PeakLR is the loudest sample of the left and of the right channel in
// frames [from, to).
func (r *Recorder) PeakLR(from, to int) (float32, float32) {
	var left, right float32
	from, to = max(from, 0), min(to, r.Frames())
	for i := from; i < to; i++ {
		left = max(left, abs(r.samples[2*i]))
		right = max(right, abs(r.samples[2*i+1]))
	}
	return left, right
}

// Reset forgets the samples recorded so far.
func (r *Recorder) Reset() {
	r.mixer.mu.Lock()
	r.samples = r.samples[:0]
	r.mixer.mu.Unlock()
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
