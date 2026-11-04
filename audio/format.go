package audio

import (
	"bufio"
	"errors"
	"io"
	"sync"
)

// ErrFormat is returned for data no registered format recognises.
var ErrFormat = errors.New("audio: unknown format")

// A Decoder yields a sound's samples, interleaved, at its own rate.
type Decoder interface {
	SampleRate() int
	Channels() int
	// Read fills p with interleaved samples and returns io.EOF at the end.
	Read(p []float32) (int, error)
	// Rewind goes back to the first sample.
	Rewind() error
}

type format struct {
	name, magic string
	decode      func(io.ReadSeeker) (Decoder, error)
}

var (
	formatsMu sync.Mutex //nolint:gochecknoglobals // guards formats
	formats   []format   //nolint:gochecknoglobals // the registry decoders add themselves to when imported
)

// RegisterFormat registers a decoder for the data starting with magic, in
// which '?' matches any byte. Decoders register themselves when imported:
//
//	import _ "github.com/merlindorin/ember/audio/wav"
func RegisterFormat(name, magic string, decode func(io.ReadSeeker) (Decoder, error)) {
	formatsMu.Lock()
	defer formatsMu.Unlock()
	formats = append(formats, format{name, magic, decode})
}

func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// NewDecoder sniffs r and returns the decoder of its format and the format's
// name.
func NewDecoder(r io.ReadSeeker) (Decoder, string, error) {
	formatsMu.Lock()
	list := formats
	formatsMu.Unlock()
	start, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, "", err
	}
	peek := bufio.NewReaderSize(r, 64)
	for _, f := range list {
		if b, perr := peek.Peek(len(f.magic)); perr != nil || !match(f.magic, b) {
			continue
		}
		if _, err = r.Seek(start, io.SeekStart); err != nil {
			return nil, "", err
		}
		var d Decoder
		d, err = f.decode(r)
		return d, f.name, err
	}
	return nil, "", ErrFormat
}
