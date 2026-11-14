package draw_test

import (
	"testing"

	"github.com/lantervale/ember/internal/draw"

	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
)

type lit struct{ li *draw.Lighting }

func (l lit) Lighting(*platform.Window) *draw.Lighting { return l.li }

func TestAWindowWithProbesReadsItsDrawnFrameBack(t *testing.T) {
	w, win := world.New(), platform.NewWindow()
	if draw.ReadsBack(w, win) {
		t.Fatal("a window nothing lights reads nothing back")
	}
	li := &draw.Lighting{}
	draw.InstallLighting(w, lit{li})
	if draw.ReadsBack(w, win) {
		t.Fatal("lights alone read nothing back")
	}
	li.Probes = []draw.Probe{{Result: &draw.ProbeResult{}}}
	if !draw.ReadsBack(w, win) {
		t.Fatal("a probe is answered from the frame drawn")
	}
}
