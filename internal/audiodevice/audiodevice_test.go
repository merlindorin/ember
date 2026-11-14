package audiodevice_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/lantervale/ember/audio"
	"github.com/lantervale/ember/internal/audiodevice"
	"github.com/lantervale/ember/world"
)

// The device itself needs sound hardware, so tests cover the byte stream
// and that nothing opens before Start.
func TestPCMEncodesTheMixAsFloat32LittleEndian(t *testing.T) {
	m := audio.NewMixer()
	clip, _ := audio.NewClip(48000, 2, []float32{0.5, -0.25, 0.5, -0.25, 0.5, -0.25})
	m.Play(clip)
	b := make([]byte, 8*2+3)
	n, err := (&audiodevice.PCM{Mixer: m}).Read(b)
	if err != nil || n != 16 {
		t.Fatal(n, err)
	}
	l := math.Float32frombits(binary.LittleEndian.Uint32(b))
	r := math.Float32frombits(binary.LittleEndian.Uint32(b[4:]))
	if l != 0.5 || r != -0.25 {
		t.Fatalf("first frame %v %v", l, r)
	}
}

func TestNothingOpensBeforeStart(t *testing.T) {
	out := audiodevice.New()
	m := audio.NewMixer()
	rec := audio.NewRecorder(m)
	w := world.New().AddPlugins(audio.Plugin{}).AddEntities(m)
	w.AddView(out)
	w.RunFrames(2)
	if out.Len() != 1 || rec.Frames() != 1600 {
		t.Fatalf("the mixer stays on its null output: %d mixers, %d frames", out.Len(), rec.Frames())
	}
	out.Close()
}
