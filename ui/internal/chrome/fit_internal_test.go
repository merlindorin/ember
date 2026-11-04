//go:build chrome

package chrome

import (
	"testing"

	"github.com/merlindorin/ember/geom"
	"github.com/stretchr/testify/assert"
)

func TestANarrowPagesPictureOpensAWindowAtLeastFiveHundredPointsEachWay(t *testing.T) {
	page := []byte("<html><head></head><body></body></html>")
	for _, c := range []struct {
		size, window geom.Vec2
	}{
		{geom.V2(160, 104), geom.V2(500, 500)},
		{geom.V2(520, 80), geom.V2(520, 80)},
		{geom.V2(420, 160), geom.V2(500, 500)},
		{geom.V2(640, 520), geom.V2(640, 520)},
	} {
		got, window := fit(page, c.size, true)
		assert.Equal(t, c.window, window, "%v", c.size)
		if window != c.size {
			assert.Contains(t, string(got), "html { width: ", "%v keeps its size on the root", c.size)
			assert.Contains(t, string(got), "position: relative", "%v", c.size)
		}
	}
}

func TestABoxDumpKeepsItsWindowAsTall(t *testing.T) {
	_, window := fit([]byte("<head></head>"), geom.V2(160, 104), false)
	assert.Equal(t, geom.V2(500, 104), window, "only pictures get the taller window")
}
