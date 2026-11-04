package audio

import (
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/merlindorin/ember/world"
)

const (
	defaultRate = 48000
	// smoothing is how long gain and pan changes take, against clicks.
	smoothing = 10 * time.Millisecond
)

// A Mixer plays every voice of a game into one stereo output. Add it to the
// World: a windowed World opens the sound device for it, and anywhere else
// (servers, tests, EMBER_HIDDEN runs) it mixes into a null output at the
// World's pace, so voices still end and events still fire.
//
//	mixer := audio.NewMixer(audio.WithBus("music", 0.6), audio.WithBus("sfx", 1))
//	ember.New().AddEntities(win, mixer)
//	mixer.Play(hit, audio.OnBus("sfx"))
type Mixer struct {
	*world.BasicEntity

	mu        sync.Mutex
	rate      int
	maxVoices int
	master    *Bus
	buses     map[string]*Bus
	order     []*Bus
	scratch   []float32
	voices    []*Voice
	music     *Voice
	serial    uint64
	events    []any
	recorder  *Recorder
	carry     float64
	pulled    atomic.Bool
}

// masterBus is the name of the bus over every other.
const masterBus = "master"

// A MixerOption sets up a Mixer.
type MixerOption func(*Mixer)

// WithBus adds a bus, a group of voices with its own volume, such as
// "music" or "sfx", and the Effects its mix runs through. "master", over
// every bus, always exists.
func WithBus(name string, volume float32, opts ...Option) MixerOption {
	set := newSettings(opts)
	return func(m *Mixer) {
		b := m.master
		if name != masterBus {
			b = &Bus{mixer: m, name: name}
			m.buses[name] = b
			m.order = append(m.order, b)
			slices.SortFunc(m.order, func(a, b *Bus) int { return strings.Compare(a.name, b.name) })
		}
		b.volume, b.effects = max(volume, 0), set.effects
	}
}

// MaxVoices caps the voices playing at once, 64 by default. Past it, a
// new sound takes the place of the oldest of the lowest priority, or is
// dropped if every voice outranks it.
func MaxVoices(n int) MixerOption { return func(m *Mixer) { m.maxVoices = max(n, 1) } }

// SampleRate is the output rate in Hz, 48000 by default.
func SampleRate(hz int) MixerOption { return func(m *Mixer) { m.rate = min(max(hz, 8000), 192000) } }

// NewMixer returns a mixer with a master bus and the buses given.
func NewMixer(opts ...MixerOption) *Mixer {
	m := &Mixer{BasicEntity: world.NewBasic(), rate: defaultRate, maxVoices: 64, buses: map[string]*Bus{}}
	m.master = &Bus{mixer: m, name: masterBus, volume: 1}
	for _, o := range opts {
		o(m)
	}
	return m
}

func (m *Mixer) GetMixer() *Mixer { return m }

func (m *Mixer) SampleRate() int { return m.rate }
func (m *Mixer) MaxVoices() int  { return m.maxVoices }

// Master is the bus over every other.
func (m *Mixer) Master() *Bus { return m.master }

// Bus returns the named bus, or nil.
func (m *Mixer) Bus(name string) *Bus {
	if name == masterBus {
		return m.master
	}
	return m.buses[name]
}

// Voices counts the voices playing: sounding, paused or fading out.
func (m *Mixer) Voices() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.voices)
}

// Music is the track PlayMusic started, or nil.
func (m *Mixer) Music() *Voice {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.music
}

// A Sound is what a voice plays: a *Clip or a *Stream. Its source is nil
// when it can't be played, such as a stream that won't rewind.
type Sound interface {
	newSource(m *Mixer, loop bool) source
}

type source interface {
	rate() int
	next(loop bool) (l, r float32, looped, ok bool)
	close()
}

// Play starts a voice. It honours Volume, Pan, Pitch, Loop, OnBus, FadeIn and
// Priority. The voice may never play when every voice outranks it.
func (m *Mixer) Play(s Sound, opts ...Option) *Voice {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.play(s, newSettings(opts))
}

// PlayMusic plays s on the "music" bus (or the one OnBus names) in place of
// the current track, crossfading with it for Crossfade.
func (m *Mixer) PlayMusic(s Sound, opts ...Option) *Voice {
	set := newSettings(opts)
	if set.bus == "" {
		set.bus = "music"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.music
	v := m.play(s, set)
	if old != nil && old.state < stopping {
		if set.crossfade > 0 {
			old.stop(m.frames(set.crossfade), true)
			v.fade = fade{from: 0, to: 1, len: m.frames(set.crossfade), power: true}
		} else {
			old.stop(m.frames(smoothing), false)
		}
	}
	m.music = v
	return v
}

// StopMusic stops the current track, over FadeOut.
func (m *Mixer) StopMusic(opts ...Option) {
	if v := m.Music(); v != nil {
		v.Stop(opts...)
	}
}

func (m *Mixer) frames(d time.Duration) int { return int(d.Seconds() * float64(m.rate)) }

func (m *Mixer) play(s Sound, set settings) *Voice {
	m.serial++
	v := &Voice{mixer: m, id: m.serial, set: set, spatialGain: 1}
	if !m.makeRoom(set.priority) {
		v.state = done
		return v
	}
	src := s.newSource(m, set.loop)
	if src == nil {
		v.state = done
		return v
	}
	v.src = src
	if s, ok := src.(*streamSource); ok {
		s.voice = v
	}
	v.step = float64(src.rate()) / float64(m.rate)
	v.bus = m.buses[set.bus]
	if set.fadeIn > 0 {
		v.fade = fade{from: 0, to: 1, len: m.frames(set.fadeIn)}
	} else {
		v.fade = fade{from: 1, to: 1}
	}
	v.gl, v.gr = v.target()
	m.voices = append(m.voices, v)
	return v
}

func (m *Mixer) makeRoom(priority int) bool {
	var victim *Voice
	active := 0
	for _, v := range m.voices {
		if v.state >= stopping {
			continue
		}
		active++
		if victim == nil || v.set.priority < victim.set.priority {
			victim = v
		}
	}
	if active < m.maxVoices {
		return true
	}
	if victim.set.priority > priority {
		return false
	}
	victim.stop(m.frames(smoothing), false)
	victim.stolen = true
	return true
}

// Read mixes the next len(p)/2 stereo frames into p, interleaved. The sound
// device calls it; once something reads the mixer, the mixer stops mixing
// into its null output.
func (m *Mixer) Read(p []float32) (int, error) {
	m.pulled.Store(true)
	m.mix(p)
	return len(p) / 2 * 2, nil
}

func (m *Mixer) mix(p []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(p)
	for _, b := range m.order {
		if len(b.effects) > 0 {
			b.buf = grow(b.buf, len(p))
		}
	}
	smooth := float32(1 / (smoothing.Seconds() * float64(m.rate)))
	live := m.voices[:0]
	for _, v := range m.voices {
		dst := p
		if v.bus != nil && len(v.bus.effects) > 0 {
			dst = v.bus.buf
		}
		if len(v.set.effects) > 0 {
			m.scratch = grow(m.scratch, len(p))
			v.mix(m.scratch, smooth)
			runEffects(v.set.effects, m.scratch, m.rate)
			for i, s := range m.scratch {
				dst[i] += s
			}
		} else {
			v.mix(dst, smooth)
		}
		if v.state == done {
			v.src.close()
			m.events = append(m.events, VoiceFinished{Voice: v, Stopped: v.stopped || v.stolen})
			if m.music == v {
				m.music = nil
			}
			continue
		}
		live = append(live, v)
	}
	clear(m.voices[len(live):])
	m.voices = live
	for _, b := range m.order {
		if len(b.effects) > 0 {
			runEffects(b.effects, b.buf, m.rate)
			for i, s := range b.buf {
				p[i] += s
			}
		}
	}
	runEffects(m.master.effects, p, m.rate)
	for i, s := range p {
		p[i] = min(max(s, -1), 1)
	}
	if m.recorder != nil {
		m.recorder.samples = append(m.recorder.samples, p[:len(p)/2*2]...)
	}
}

// grow returns buf cleared at length n, reusing its memory.
func grow(buf []float32, n int) []float32 {
	if cap(buf) < n {
		return make([]float32, n)
	}
	buf = buf[:n]
	clear(buf)
	return buf
}

// advance mixes dt seconds into the null output, unless a device reads.
func (m *Mixer) advance(dt float32) {
	if m.pulled.Load() {
		return
	}
	m.carry += float64(dt) * float64(m.rate)
	n := int(m.carry)
	m.carry -= float64(n)
	if n > 0 {
		m.mix(make([]float32, 2*n))
	}
}

func (m *Mixer) drain() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	ev := m.events
	m.events = nil
	return ev
}

// VoiceFinished is emitted when a voice ends, or is stopped (Stopped).
type VoiceFinished struct {
	Voice   *Voice
	Stopped bool
}

// VoiceLooped is emitted each time a looping voice starts over.
type VoiceLooped struct{ Voice *Voice }

// A Bus groups voices under one volume.
type Bus struct {
	mixer   *Mixer
	name    string
	volume  float32
	muted   bool
	effects []Effect
	buf     []float32
}

// Effects are what the bus's mix runs through, in order.
func (b *Bus) Effects() []Effect {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	return slices.Clone(b.effects)
}

// SetEffects replaces the bus's effects; a reverb dropped stops ringing.
func (b *Bus) SetEffects(e ...Effect) {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	b.effects = slices.Clone(e)
}

func (b *Bus) Name() string { return b.name }

func (b *Bus) Volume() float32 {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	return b.volume
}

func (b *Bus) SetVolume(v float32) {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	b.volume = max(v, 0)
}

func (b *Bus) Muted() bool {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	return b.muted
}

func (b *Bus) SetMuted(muted bool) {
	b.mixer.mu.Lock()
	defer b.mixer.mu.Unlock()
	b.muted = muted
}

func (b *Bus) gain() float32 {
	if b == nil {
		return 1
	}
	if b.muted {
		return 0
	}
	return b.volume
}
