package light //nolint:testpackage // white-box: shares light's test scene, reaches unexported state

import (
	"testing"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/twod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lightedSprite is what a frame draws of a sprite of opts next to a lamp:
// the sprite's vertices, the occluders, the glows and the silhouettes.
type lightedSprite struct {
	front, occluders, emissive []draw.Vertex
	silhouettes                []draw.SilhouetteQuad
}

// lighted draws a sprite next to a lamp, given opts as it is made when
// live is false, else applied one frame after it was added.
func lighted(t *testing.T, live bool, opts ...twod.Option) lightedSprite {
	t.Helper()
	w, win, _ := scene(t)
	var made []twod.SpriteOption
	if !live {
		for _, o := range opts {
			made = append(made, o)
		}
	}
	sp := twod.NewSprite(picture(10, 20), append([]twod.SpriteOption{twod.At(-20, 0)}, made...)...)
	layer := twod.NewLayer(0, Lit()).Add(sp, NewPoint(80, color.White, At(10, 0)))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	if live {
		for _, o := range opts {
			o(sp)
		}
	}
	w.RunFrames(2)
	l := draw.LightingFor(w, win)
	require.NotNil(t, l)
	return lightedSprite{
		front:       append([]draw.Vertex(nil), draw.FrontFor(w, win).Vertices...),
		occluders:   append([]draw.Vertex(nil), l.Occluders.Vertices...),
		emissive:    append([]draw.Vertex(nil), l.Emissive.Vertices...),
		silhouettes: append([]draw.SilhouetteQuad(nil), l.SilhouetteQuads...),
	}
}

// PerPixel is how a Lit layer's sprites are lit anyway: it shows only that
// it is applied the same.
func TestEverySpriteOptionAppliesToALiveSpriteTheNextFrame(t *testing.T) {
	for name, opts := range map[string][]twod.Option{
		"Occluder":            {Occluder(1)},
		"Occluder and Height": {Occluder(), Height(8)},
		"Silhouette":          {Silhouette()},
		"Emissive":            {Emissive(12)},
		"Unlit":               {Unlit()},
		"PerPixel":            {PerPixel()},
		"LitAt":               {LitAt(0, 10)},
		"ShadedBy":            {ShadedBy(1)},
	} {
		t.Run(name, func(t *testing.T) {
			made, live := lighted(t, false, opts...), lighted(t, true, opts...)
			assert.Equal(t, made, live, "the same as when made with it")
			if name != "PerPixel" {
				assert.NotEqual(t, lighted(t, false), made, "and it changes what draws")
			}
		})
	}
}

func TestSettersSwitchALiveSpritesShadow(t *testing.T) {
	w, win, _ := scene(t)
	orc := twod.NewSprite(picture(10, 20), twod.At(-20, 0))
	layer := twod.NewLayer(0, Lit()).Add(orc, NewPoint(80, color.White, At(10, 0)))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	l := func() *draw.Lighting { return draw.LightingFor(w, win) }

	SetSilhouette(orc)
	w.RunFrames(1)
	assert.NotEmpty(t, l().SilhouetteQuads, "set after Add, it casts its picture the next frame")
	assert.Empty(t, l().Occluders.Vertices)

	SetOccluder(orc, 2)
	SetHeight(orc, 8)
	w.RunFrames(1)
	assert.Empty(t, l().SilhouetteQuads)
	require.Len(t, l().Occluders.Vertices, 4, "back to its box")
	assert.Equal(t, draw.OccluderColor(1<<2, 16), l().Occluders.Vertices[0].Color, "of group 2, 8 tall, zoomed twice")
	groups, on := OccluderOf(orc)
	assert.Equal(t, []int{2}, groups)
	assert.True(t, on)
	assert.Equal(t, float32(8), HeightOf(orc))

	RemoveOccluder(orc)
	w.RunFrames(1)
	assert.Empty(t, l().Occluders.Vertices)
	assert.Empty(t, l().SilhouetteQuads)

	SetEmissive(orc, 6)
	w.RunFrames(1)
	assert.NotEmpty(t, l().Emissive.Vertices, "glows")
	assert.Equal(t, float32(6), EmissiveOf(orc))
	SetEmissive(orc, 0)
	w.RunFrames(1)
	assert.Empty(t, l().Emissive.Vertices)
}
