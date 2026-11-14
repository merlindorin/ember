// Package vorbis decodes Ogg Vorbis sounds, mono or stereo, through
// github.com/jfreymuth/oggvorbis, and registers them with package audio.
package vorbis

import (
	"errors"
	"fmt"
	"io"

	"github.com/jfreymuth/oggvorbis"
	"github.com/lantervale/ember/audio"
)

func init() { //nolint:gochecknoinits // importing the package registers the decoder, by design
	audio.RegisterFormat("vorbis", "OggS", func(r io.ReadSeeker) (audio.Decoder, error) { return NewDecoder(r) })
}

// Decoder reads the samples of an Ogg Vorbis stream.
type Decoder struct {
	r *oggvorbis.Reader
}

// NewDecoder reads the headers of an Ogg Vorbis stream.
func NewDecoder(r io.ReadSeeker) (d *Decoder, err error) { //nolint:nonamedreturns // recoverInto sets err
	defer recoverInto(&err)
	or, err := oggvorbis.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("vorbis: %w", err)
	}
	if c := or.Channels(); c < 1 || c > 2 {
		return nil, fmt.Errorf("vorbis: %d channels, only mono and stereo are supported", c)
	}
	return &Decoder{r: or}, nil
}

func (d *Decoder) SampleRate() int { return d.r.SampleRate() }
func (d *Decoder) Channels() int   { return d.r.Channels() }

func (d *Decoder) Read(p []float32) (n int, err error) { //nolint:nonamedreturns // recoverInto sets err
	defer recoverInto(&err)
	n, err = d.r.Read(p)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = io.EOF
	}
	return n, err
}

func (d *Decoder) Rewind() (err error) { //nolint:nonamedreturns // recoverInto sets err
	defer recoverInto(&err)
	return d.r.SetPosition(0)
}

// recoverInto turns a panic of the decoder on corrupt data into an error.
func recoverInto(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("vorbis: corrupt stream: %v", r)
	}
}
