package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// maxClipSamples bounds a decoded clip: ten minutes of 48 kHz stereo.
const maxClipSamples = 10 * 60 * 48000 * 2

// A Clip is a sound decoded whole into memory: effects and short loops.
// Any number of voices play one clip at once.
type Clip struct {
	rate, channels int
	samples        []float32
}

// NewClip makes a clip from interleaved samples, one or two channels, as
// games that synthesise their sounds do.
func NewClip(rate, channels int, samples []float32) (*Clip, error) {
	if rate <= 0 || channels < 1 || channels > 2 {
		return nil, fmt.Errorf("audio: clip of %d channels at %d Hz", channels, rate)
	}
	return &Clip{rate: rate, channels: channels, samples: samples[:len(samples)/channels*channels]}, nil
}

// Load decodes the file name from fsys into a Clip.
func Load(fsys fs.FS, name string) (*Clip, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	c, err := Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("audio: %s: %w", name, err)
	}
	return c, nil
}

// Decode reads a whole sound of any registered format.
func Decode(r io.ReadSeeker) (*Clip, error) {
	d, _, err := NewDecoder(r)
	if err != nil {
		return nil, err
	}
	return ReadAll(d)
}

// ReadAll decodes what is left of d into a Clip.
func ReadAll(d Decoder) (*Clip, error) {
	var samples []float32
	buf := make([]float32, 4096)
	for empty := 0; ; {
		n, err := d.Read(buf)
		samples = append(samples, buf[:n]...)
		if len(samples) > maxClipSamples {
			return nil, errors.New("audio: clip too long, open it as a stream")
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if empty = (empty + 1) * boolInt(n == 0); empty > maxEmptyReads {
			return nil, io.ErrNoProgress
		}
	}
	return NewClip(d.SampleRate(), d.Channels(), samples)
}

func (c *Clip) SampleRate() int { return c.rate }
func (c *Clip) Channels() int   { return c.channels }

// Samples are the clip's interleaved samples.
func (c *Clip) Samples() []float32 { return c.samples }

// Frames is the clip's length in frames, one sample per channel.
func (c *Clip) Frames() int { return len(c.samples) / c.channels }

func (c *Clip) Duration() time.Duration {
	return time.Duration(c.Frames()) * time.Second / time.Duration(c.rate)
}

func (c *Clip) newSource(*Mixer, bool) source { return &clipSource{clip: c} }

type clipSource struct {
	clip *Clip
	at   int
}

func (s *clipSource) rate() int { return s.clip.rate }

func (s *clipSource) next(loop bool) (float32, float32, bool, bool) {
	c := s.clip
	looped := false
	if s.at >= len(c.samples) {
		if !loop || len(c.samples) == 0 {
			return 0, 0, false, false
		}
		s.at, looped = 0, true
	}
	l := c.samples[s.at]
	r := l
	if c.channels == 2 {
		r = c.samples[s.at+1]
	}
	s.at += c.channels
	return l, r, looped, true
}

func (s *clipSource) close() {}
