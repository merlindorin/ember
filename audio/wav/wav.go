// Package wav decodes RIFF WAVE sounds (PCM 8, 16, 24 and 32-bit, and
// 32-bit float, mono or stereo) and registers them with package audio.
package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/merlindorin/ember/audio"
)

func init() { //nolint:gochecknoinits // importing the package registers the decoder, by design
	audio.RegisterFormat("wav", "RIFF????WAVE", func(r io.ReadSeeker) (audio.Decoder, error) { return NewDecoder(r) })
}

const (
	formatPCM        = 1
	formatFloat      = 3
	formatExtensible = 0xfffe
)

// Decoder reads the samples of a WAVE file.
type Decoder struct {
	r              io.ReadSeeker
	rate, channels int
	bits           int
	float          bool
	start, size    int64
	at             int64
	buf            []byte
}

// NewDecoder reads the header of a WAVE file and stops at its samples.
func NewDecoder(r io.ReadSeeker) (*Decoder, error) {
	var riff [12]byte
	if _, err := io.ReadFull(r, riff[:]); err != nil {
		return nil, fmt.Errorf("wav: %w", err)
	}
	if string(riff[:4]) != "RIFF" || string(riff[8:]) != "WAVE" {
		return nil, errors.New("wav: not a RIFF WAVE file")
	}
	d := &Decoder{r: r}
	pos := int64(12)
	haveFormat := false
	for {
		var h [8]byte
		if _, err := io.ReadFull(r, h[:]); err != nil {
			return nil, fmt.Errorf("wav: no data chunk: %w", err)
		}
		pos += 8
		id, size := string(h[:4]), int64(binary.LittleEndian.Uint32(h[4:]))
		switch id {
		case "fmt ":
			if err := d.readFormat(r, size); err != nil {
				return nil, err
			}
			haveFormat = true
		case "data":
			if !haveFormat {
				return nil, errors.New("wav: data before fmt")
			}
			d.start, d.size = pos, size-size%int64(d.frameSize())
			return d, nil
		default:
			if _, err := r.Seek(size+size&1, io.SeekCurrent); err != nil {
				return nil, fmt.Errorf("wav: %w", err)
			}
		}
		pos += size + size&1
	}
}

func (d *Decoder) readFormat(r io.Reader, size int64) error {
	if size < 16 || size > 1<<16 {
		return fmt.Errorf("wav: fmt chunk of %d bytes", size)
	}
	b := make([]byte, size+size&1)
	if _, err := io.ReadFull(r, b); err != nil {
		return fmt.Errorf("wav: %w", err)
	}
	tag := binary.LittleEndian.Uint16(b)
	d.channels = int(binary.LittleEndian.Uint16(b[2:]))
	d.rate = int(binary.LittleEndian.Uint32(b[4:]))
	d.bits = int(binary.LittleEndian.Uint16(b[14:]))
	if tag == formatExtensible && size >= 26 {
		tag = binary.LittleEndian.Uint16(b[24:])
	}
	switch {
	case tag == formatPCM && (d.bits == 8 || d.bits == 16 || d.bits == 24 || d.bits == 32):
	case tag == formatFloat && d.bits == 32:
		d.float = true
	default:
		return fmt.Errorf("wav: unsupported format %#x of %d bits", tag, d.bits)
	}
	if d.channels < 1 || d.channels > 2 {
		return fmt.Errorf("wav: %d channels, only mono and stereo are supported", d.channels)
	}
	if d.rate < 1000 || d.rate > 384000 {
		return fmt.Errorf("wav: sample rate %d", d.rate)
	}
	return nil
}

func (d *Decoder) frameSize() int { return d.channels * d.bits / 8 }

func (d *Decoder) SampleRate() int { return d.rate }
func (d *Decoder) Channels() int   { return d.channels }

// Frames is the length of the sound in frames.
func (d *Decoder) Frames() int64 { return d.size / int64(d.frameSize()) }

func (d *Decoder) Rewind() error {
	_, err := d.r.Seek(d.start, io.SeekStart)
	d.at = 0
	return err
}

func (d *Decoder) Read(p []float32) (int, error) {
	width := d.bits / 8
	n := min(int64(len(p))/int64(d.channels)*int64(d.channels), (d.size-d.at)/int64(width))
	if n <= 0 {
		if len(p) < d.channels {
			return 0, nil
		}
		return 0, io.EOF
	}
	need := int(n) * width
	if cap(d.buf) < need {
		d.buf = make([]byte, need)
	}
	b := d.buf[:need]
	got, err := io.ReadFull(d.r, b)
	got -= got % (width * d.channels)
	d.at += int64(got)
	samples := got / width
	for i := range samples {
		p[i] = d.sample(b[i*width:])
	}
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		d.size = d.at
		if samples == 0 {
			return 0, io.EOF
		}
		err = nil
	}
	return samples, err
}

func (d *Decoder) sample(b []byte) float32 {
	switch {
	case d.float:
		return float32(math.Float32frombits(binary.LittleEndian.Uint32(b)))
	case d.bits == 8:
		return float32(int(b[0])-128) / 128
	case d.bits == 16:
		return float32(int16(binary.LittleEndian.Uint16(b))) / (1 << 15) //nolint:gosec // signed samples, read by their bits
	case d.bits == 24:
		v := uint32(b[0])<<8 | uint32(b[1])<<16 | uint32(b[2])<<24
		return float32(int32(v)>>8) / (1 << 23) //nolint:gosec // signed samples, read by their bits
	}
	v := int32(binary.LittleEndian.Uint32(b)) //nolint:gosec // signed samples, read by their bits
	return float32(float64(v) / (1 << 31))
}

// Encode writes c as a 16-bit PCM WAVE file.
func Encode(w io.Writer, c *audio.Clip) error {
	samples := c.Samples()
	data := len(samples) * 2
	rate, channels := c.SampleRate(), c.Channels()
	if data > math.MaxUint32-36 || rate > math.MaxUint32/4 || rate < 0 || channels < 0 || channels > 2 {
		return fmt.Errorf("wav: %d samples of %d channels at %d Hz do not fit a WAVE file", len(samples), channels, rate)
	}
	h := make([]byte, 44, 44+data)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+data))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], formatPCM)
	binary.LittleEndian.PutUint16(h[22:], uint16(channels))
	binary.LittleEndian.PutUint32(h[24:], uint32(rate))
	binary.LittleEndian.PutUint32(h[28:], uint32(rate*channels*2)) //nolint:gosec // rate and channels are bounded above
	binary.LittleEndian.PutUint16(h[32:], uint16(channels*2))
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(data))
	for _, s := range samples {
		v := int16(math.Round(float64(min(max(s, -1), 1)) * 32767))
		h = binary.LittleEndian.AppendUint16(h, uint16(v)) //nolint:gosec // signed samples, stored by their bits
	}
	_, err := w.Write(h)
	return err
}
