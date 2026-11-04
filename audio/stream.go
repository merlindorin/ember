package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
)

const (
	chunkSamples = 4096
	chunksAhead  = 16
)

// A Stream is a sound decoded as it plays: music and long ambiences. One
// voice plays a stream at a time; playing it again restarts it.
//
// Under a device, a goroutine decodes ahead of the audio thread; with the
// null output or a Recorder, the stream decodes as it is mixed, so tests
// stay deterministic.
type Stream struct {
	dec    Decoder
	file   io.Closer
	active *streamSource
}

// Open opens the file name from fsys as a Stream; Close releases it.
func Open(fsys fs.FS, name string) (*Stream, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		var b []byte
		b, err = io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		rs, f = bytes.NewReader(b), nil
	}
	d, _, err := NewDecoder(rs)
	if err != nil {
		if f != nil {
			f.Close()
		}
		return nil, fmt.Errorf("audio: %s: %w", name, err)
	}
	s := NewStream(d)
	if f != nil {
		s.file = f
	}
	return s, nil
}

// NewStream streams from d.
func NewStream(d Decoder) *Stream { return &Stream{dec: d} }

func (s *Stream) SampleRate() int { return s.dec.SampleRate() }
func (s *Stream) Channels() int   { return s.dec.Channels() }

// Close stops the stream's voice and closes its file.
func (s *Stream) Close() error {
	if src := s.active; src != nil {
		src.mixer.mu.Lock()
		src.cancel()
		src.mixer.mu.Unlock()
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *Stream) newSource(m *Mixer, loop bool) source {
	if s.active != nil {
		s.active.cancel()
	}
	if s.dec.Rewind() != nil {
		return nil
	}
	src := &streamSource{stream: s, mixer: m, loop: loop, channels: s.dec.Channels()}
	s.active = src
	return src
}

type chunk struct {
	samples []float32
	looped  bool
	end     bool
}

type streamSource struct {
	stream   *Stream
	mixer    *Mixer
	voice    *Voice
	loop     bool
	channels int

	buf       []float32
	at        int
	looped    bool
	loopNext  bool
	ended     bool
	cancelled bool

	chunks chan chunk
	done   chan struct{}
	exited chan struct{}
}

func (s *streamSource) rate() int { return s.stream.dec.SampleRate() }

func (s *streamSource) next(bool) (float32, float32, bool, bool) {
	if s.cancelled {
		return 0, 0, false, false
	}
	if s.at+s.channels > len(s.buf) {
		if s.ended {
			return 0, 0, false, false
		}
		if !s.refill() {
			return 0, 0, false, true
		}
		if s.at+s.channels > len(s.buf) {
			return s.next(false)
		}
	}
	l := s.buf[s.at]
	r := l
	if s.channels == 2 {
		r = s.buf[s.at+1]
	}
	s.at += s.channels
	looped := s.looped
	s.looped = false
	return l, r, looped, true
}

// refill reports false on an underrun, when the decoding goroutine is late.
func (s *streamSource) refill() bool {
	if s.chunks == nil && s.mixer.pulled.Load() {
		s.chunks, s.done, s.exited = make(chan chunk, chunksAhead), make(chan struct{}), make(chan struct{})
		go s.decodeAhead()
	}
	var c chunk
	if s.chunks != nil {
		select {
		case c = <-s.chunks:
		default:
			return false
		}
	} else {
		c = s.decode(make([]float32, chunkSamples))
	}
	s.buf, s.at, s.ended, s.looped = c.samples, 0, c.end, c.looped
	return true
}

// maxEmptyReads is how many reads in a row may return nothing, as decoders
// do between pages, before a decoder is taken as stuck.
const maxEmptyReads = 100

func readFull(d Decoder, p []float32) (int, error) {
	n, empty := 0, 0
	for n < len(p) {
		k, err := d.Read(p[n:])
		n += k
		if err != nil {
			return n, err
		}
		if empty = (empty + 1) * boolInt(k == 0); empty > maxEmptyReads {
			return n, io.ErrNoProgress
		}
	}
	return n, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// decode reads the next chunk; a chunk marked looped starts at the loop
// point.
func (s *streamSource) decode(buf []float32) chunk {
	d := s.stream.dec
	c := chunk{looped: s.loopNext}
	s.loopNext = false
	for try := 0; try < 2; try++ {
		n, err := readFull(d, buf)
		c.samples = buf[:n/s.channels*s.channels]
		if err == nil {
			return c
		}
		if !errors.Is(err, io.EOF) || !s.loop || d.Rewind() != nil {
			c.end = true
			return c
		}
		if len(c.samples) > 0 {
			s.loopNext = true
			return c
		}
		c.looped = true
	}
	c.end = true
	return c
}

func (s *streamSource) decodeAhead() {
	defer close(s.exited)
	for {
		c := s.decode(make([]float32, chunkSamples))
		select {
		case s.chunks <- c:
		case <-s.done:
			return
		}
		if c.end {
			return
		}
	}
}

func (s *streamSource) cancel() {
	if s.cancelled {
		return
	}
	s.cancelled = true
	if s.voice != nil && s.voice.state < done {
		s.voice.state, s.voice.stopped = done, true
	}
	s.close()
}

func (s *streamSource) close() {
	if s.done != nil {
		close(s.done)
		<-s.exited
		s.done = nil
	}
	if s.stream.active == s {
		s.stream.active = nil
	}
}
