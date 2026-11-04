// Package audiodevice plays a World's mixer on the system's sound output
// through oto: CoreAudio on macOS and WASAPI on Windows without cgo, ALSA on
// Linux with it, Web Audio in a page. The desktop and web runners start it;
// servers and tests never import it.
package audiodevice

import (
	"encoding/binary"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/world"
)

// latency is how much sound the device buffers ahead.
const latency = 40 * time.Millisecond

//nolint:gochecknoglobals // a process has one sound device, made once
var (
	contextOnce sync.Once
	context     *oto.Context
	contextRate int
	// live is the context once made, for Resume.
	live atomic.Pointer[oto.Context]
)

// Output is a view of a World's mixers that plays the first one once
// started. A process has one sound device, so other mixers stay on their
// null output.
type Output struct {
	world.Group[*audio.Mixer]
	started bool
	mixer   *audio.Mixer
	player  *oto.Player
}

func New() *Output { return &Output{} }

func (o *Output) AddByInterface(e world.Identifier) {
	o.Group.AddByInterface(e)
	if o.started {
		o.attach()
	}
}

// Start plays the first mixer, now or when one is added.
func (o *Output) Start() {
	o.started = true
	o.attach()
}

// Resume starts the sound device if the browser held it until a click
// or key; a page's runner calls it from the canvas's click, key and touch
// handlers, which may.
func (o *Output) Resume() {
	if c := live.Load(); c != nil {
		if err := c.Resume(); err != nil {
			log.Printf("audio: resuming the sound device: %v", err)
		}
	}
}

// Close stops the sound.
func (o *Output) Close() {
	o.started = false
	if o.player != nil {
		o.player.Close()
		o.player = nil
	}
}

func (o *Output) attach() {
	if o.player != nil {
		return
	}
	for m := range o.All() {
		o.play(m)
		return
	}
}

func (o *Output) play(m *audio.Mixer) {
	contextOnce.Do(func() {
		ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   m.SampleRate(),
			ChannelCount: 2,
			Format:       oto.FormatFloat32LE,
			BufferSize:   latency,
		})
		if err != nil {
			log.Printf("audio: no sound device, playing silently: %v", err)
			return
		}
		// A page's context starts suspended and outputs silence until the
		// player's first click or key resumes it, so the players are made
		// at once rather than after that click.
		if runtime.GOOS != "js" {
			<-ready
		}
		context, contextRate = ctx, m.SampleRate()
		live.Store(ctx)
	})
	if context == nil {
		return
	}
	if contextRate != m.SampleRate() {
		log.Printf("audio: the device runs at %d Hz, not %d; this mixer plays silently", contextRate, m.SampleRate())
		return
	}
	o.mixer = m
	o.player = context.NewPlayer(&PCM{Mixer: m})
	o.player.SetBufferSize(int(latency.Seconds()*float64(contextRate)) * 2 * 4)
	o.player.Play()
}

// PCM reads a mixer as little-endian float32 stereo bytes.
type PCM struct {
	Mixer *audio.Mixer
	buf   []float32
}

func (p *PCM) Read(b []byte) (int, error) {
	n := len(b) / 8 * 2
	if cap(p.buf) < n {
		p.buf = make([]float32, n)
	}
	s := p.buf[:n]
	_, _ = p.Mixer.Read(s) // a mixer always fills s, silence past its sounds
	for i, v := range s {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
	}
	return 4 * n, nil
}
