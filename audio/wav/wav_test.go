package wav_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/merlindorin/ember/audio"
	"github.com/merlindorin/ember/audio/wav"
)

func file(tag, channels, rate, bits int, data []byte, extra ...[]byte) []byte {
	var fmtc []byte
	fmtc = binary.LittleEndian.AppendUint16(fmtc, uint16(tag))
	fmtc = binary.LittleEndian.AppendUint16(fmtc, uint16(channels))
	fmtc = binary.LittleEndian.AppendUint32(fmtc, uint32(rate))
	fmtc = binary.LittleEndian.AppendUint32(fmtc, uint32(rate*channels*bits/8))
	fmtc = binary.LittleEndian.AppendUint16(fmtc, uint16(channels*bits/8))
	fmtc = binary.LittleEndian.AppendUint16(fmtc, uint16(bits))
	if tag == 0xfffe {
		fmtc = binary.LittleEndian.AppendUint16(fmtc, 22)
		fmtc = binary.LittleEndian.AppendUint16(fmtc, uint16(bits))
		fmtc = binary.LittleEndian.AppendUint32(fmtc, 3)
		fmtc = binary.LittleEndian.AppendUint16(fmtc, 3)
		fmtc = append(fmtc, make([]byte, 14)...)
	}
	chunk := func(id string, b []byte) []byte {
		out := append([]byte(id), binary.LittleEndian.AppendUint32(nil, uint32(len(b)))...)
		out = append(out, b...)
		if len(b)%2 == 1 {
			out = append(out, 0)
		}
		return out
	}
	body := []byte("WAVE")
	body = append(body, chunk("fmt ", fmtc)...)
	for _, e := range extra {
		body = append(body, chunk("LIST", e)...)
	}
	body = append(body, chunk("data", data)...)
	return append(append([]byte("RIFF"), binary.LittleEndian.AppendUint32(nil, uint32(len(body)))...), body...)
}

func decode(t *testing.T, b []byte) *audio.Clip {
	t.Helper()
	c, err := audio.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func same(t *testing.T, got, want []float32, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%d samples, want %d", len(got), len(want))
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > tol {
			t.Fatalf("sample %d: %v, want %v", i, got[i], want[i])
		}
	}
}

func TestEncodeRoundTrips(t *testing.T) {
	want := []float32{0, 0.5, -0.5, 1, -1, 0.25}
	c, _ := audio.NewClip(22050, 2, want)
	var buf bytes.Buffer
	if err := wav.Encode(&buf, c); err != nil {
		t.Fatal(err)
	}
	got := decode(t, buf.Bytes())
	if got.SampleRate() != 22050 || got.Channels() != 2 || got.Frames() != 3 {
		t.Fatalf("%d Hz, %d channels, %d frames", got.SampleRate(), got.Channels(), got.Frames())
	}
	same(t, got.Samples(), want, 1.0/32767)
}

func TestEverySampleFormat(t *testing.T) {
	f32 := binary.LittleEndian.AppendUint32(nil, math.Float32bits(0.75))
	f32 = binary.LittleEndian.AppendUint32(f32, math.Float32bits(-0.25))
	for _, c := range []struct {
		name string
		file []byte
		want []float32
	}{
		{"8-bit", file(1, 1, 8000, 8, []byte{128, 255, 0}), []float32{0, 127.0 / 128, -1}},
		{"24-bit", file(1, 1, 8000, 24, []byte{0, 0, 0x40, 0, 0, 0xc0}), []float32{0.5, -0.5}},
		{"32-bit", file(1, 1, 8000, 32, []byte{0, 0, 0, 0x40, 0, 0, 0, 0xc0}), []float32{0.5, -0.5}},
		{"float", file(3, 2, 8000, 32, f32), []float32{0.75, -0.25}},
		{"extensible", file(0xfffe, 2, 8000, 32, f32), []float32{0.75, -0.25}},
		{"skips chunks", file(1, 1, 8000, 8, []byte{192}, []byte("odd")), []float32{0.5}},
	} {
		t.Run(c.name, func(t *testing.T) { same(t, decode(t, c.file).Samples(), c.want, 1e-6) })
	}
}

func TestRejectsWhatItCannotPlay(t *testing.T) {
	for name, b := range map[string][]byte{
		"adpcm":    file(2, 1, 8000, 4, []byte{1}),
		"surround": file(1, 6, 8000, 16, make([]byte, 12)),
		"no rate":  file(1, 1, 0, 16, make([]byte, 2)),
		"riff":     []byte("RIFF\x04\x00\x00\x00AVI "),
	} {
		if _, err := wav.NewDecoder(bytes.NewReader(b)); err == nil {
			t.Fatalf("%s decoded", name)
		}
	}
}

func TestRewindAndTruncation(t *testing.T) {
	data := make([]byte, 2000)
	for i := range data {
		data[i] = byte(i)
	}
	b := file(1, 2, 8000, 16, data)
	d, err := wav.NewDecoder(bytes.NewReader(b))
	if err != nil || d.Frames() != 500 {
		t.Fatal(d.Frames(), err)
	}
	first, _ := audio.ReadAll(d)
	if err = d.Rewind(); err != nil {
		t.Fatal(err)
	}
	again, _ := audio.ReadAll(d)
	same(t, again.Samples(), first.Samples(), 0)

	for n := range len(b) {
		cut, cerr := wav.NewDecoder(bytes.NewReader(b[:n]))
		if cerr != nil {
			continue
		}
		c, cerr := audio.ReadAll(cut)
		if cerr != nil || len(c.Samples())%2 != 0 || len(c.Samples()) > 1000 {
			t.Fatalf("truncated at %d: %v", n, cerr)
		}
	}
}

func TestReadKeepsWholeFrames(t *testing.T) {
	d, _ := wav.NewDecoder(bytes.NewReader(file(1, 2, 8000, 16, make([]byte, 40))))
	p := make([]float32, 3)
	n, err := d.Read(p)
	if n != 2 || err != nil {
		t.Fatal(n, err)
	}
	for {
		n, err = d.Read(p)
		if errors.Is(err, io.EOF) {
			break
		}
		if n%2 != 0 {
			t.Fatal("split a frame")
		}
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(file(1, 2, 44100, 16, make([]byte, 64)))
	f.Add(file(3, 1, 8000, 32, make([]byte, 16)))
	f.Add(file(0xfffe, 2, 8000, 24, make([]byte, 18)))
	f.Fuzz(func(_ *testing.T, b []byte) {
		d, err := wav.NewDecoder(bytes.NewReader(b))
		if err != nil {
			return
		}
		p := make([]float32, 1024)
		for range 1 << 12 {
			if _, err = d.Read(p); err != nil {
				return
			}
		}
	})
}
