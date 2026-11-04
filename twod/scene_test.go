package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/node/node3d"
	"github.com/merlindorin/ember/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTextureDecodesAPNG(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 4))))
	fsys := fstest.MapFS{"crate.png": {Data: buf.Bytes()}}
	tex, err := LoadTexture(fsys, "crate.png", Pixelated())
	require.NoError(t, err)
	assert.Equal(t, geom.V2(8, 4), tex.Size())
	assert.Equal(t, render.FilterNearest, tex.Filter)
	_, err = LoadTexture(fsys, "missing.png")
	assert.Error(t, err)
	_, err = LoadTexture(fstest.MapFS{"bad.png": {Data: []byte("nope")}}, "bad.png")
	assert.Error(t, err)
}

func level() *Layer {
	sheet := atlas(32, 16)
	root := NewLayer(0)
	own := func(parent *node.Node, e node.Noded) {
		parent.AddNode(e)
		_ = e.GetSceneNode().SetOwner(&root.Node)
	}
	ground := NewLayer(0, SortByY(), Tint(color.RGB(1, 0.85, 0.7)))
	ground.SetName("Ground")
	own(&root.Node, ground)
	for i, at := range []geom.Vec2{geom.V2(120, 200), geom.V2(300, 150), geom.V2(480, 260)} {
		own(
			&ground.Node,
			NewSprite(
				sheet,
				At(at.X, at.Y),
				Still(Strip(sheet, Cell(16, 16))[i%2]),
				Scale(2, 2),
				Rotation(float32(i)*0.3),
			),
		)
	}
	hero := walker()
	hero.SetName("Hero")
	Playing("walk").sprite(hero)
	At(250, 220)(hero)
	own(&ground.Node, hero)
	own(
		&root.Node,
		NewLine(
			[]geom.Vec2{geom.V2(0, 300), geom.V2(200, 260), geom.V2(400, 330)},
			Width(6),
			Joints(RoundJoint),
			Caps(Flat, RoundCap),
			Color(color.RGB(0.6, 0.5, 0.3)),
			Blend(Screen),
		),
	)
	own(
		&root.Node,
		NewPolygon(
			[]geom.Vec2{
				geom.V2(0, 0),
				geom.V2(80, 0),
				geom.V2(80, 30),
				geom.V2(30, 30),
				geom.V2(30, 60),
				geom.V2(0, 60),
			},
			At(600, 100),
			Color(color.RGB(0.2, 0.4, 0.8)),
		),
	)
	hud := NewLayer(5, Fixed())
	own(&root.Node, hud)
	own(&hud.Node, NewText("Wood 12", At(16, 12), FontSize(18)))
	return root
}

func snapshot(t *testing.T, root Entity) []draw.Vertex {
	t.Helper()
	w, win, c := app()
	win.AppendChild(root)
	w.AddEntities(root)
	w.RunFrames(1)
	return append([]draw.Vertex(nil), c.Unindexed()...)
}

func TestA2DLevelRoundTripsThroughAScene(t *testing.T) {
	original := level()
	want := snapshot(t, original)
	require.NotEmpty(t, want)

	s, err := node3d.Pack(original, nil)
	require.NoError(t, err)
	types := map[string]bool{}
	for _, r := range s.Resources {
		types[r.Type] = true
	}
	assert.Equal(t, map[string]bool{"twod.Texture": true, "twod.Animations": true}, types)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	reloaded, err := node3d.Instantiate(back, nil)
	require.NoError(t, err)
	root, ok := reloaded.(*Layer)
	require.True(t, ok, "the root is a twod.Layer: %T", reloaded)

	got := snapshot(t, root)
	require.Len(t, got, len(want))
	for i := range want {
		assert.InDelta(t, want[i].Pos.X, got[i].Pos.X, 1e-3, "vertex %d", i)
		assert.InDelta(t, want[i].Pos.Y, got[i].Pos.Y, 1e-3, "vertex %d", i)
		assert.Equal(t, want[i].Color, got[i].Color, "vertex %d", i)
		assert.Equal(t, want[i].UV, got[i].UV, "vertex %d", i)
	}
	hero, ok := node.Find[*Sprite](root.GetSceneNode(), "Ground/Hero")
	require.True(t, ok)
	name, playing := hero.Playing()
	assert.Equal(t, "walk", name)
	assert.True(t, playing, "private fields are scene properties")
	assert.Contains(t, string(data), `"screen"`, "blend modes are saved as words")
	for _, k := range root.GetSceneNode().ChildNodes() {
		if l, isLine := k.Entity().(*Line); isLine {
			assert.Equal(t, Screen, l.Blend())
		}
	}
}

func TestParticlesRoundTripThroughAScene(t *testing.T) {
	root := NewLayer(0)
	fire := NewParticles(
		atlas(4, 4),
		Rate(25),
		Lifetime(0.5, 1),
		Gravity(geom.V2(0, -20)),
		ColorOverLife(color.Red, color.RGBA(0, 0, 0, 0)),
		Seed(3),
		LocalSpace(),
	)
	root.AddNode(fire)
	_ = fire.GetSceneNode().SetOwner(&root.Node)
	s, err := node3d.Pack(root, nil)
	require.NoError(t, err)
	data, err := node.Encode(s)
	require.NoError(t, err)
	back, err := node.Decode(data)
	require.NoError(t, err)
	reloaded, err := node3d.Instantiate(back, nil)
	require.NoError(t, err)
	got, ok := node.Find[*Particles](reloaded.GetSceneNode(), "particles")
	require.True(t, ok)
	assert.Equal(t, fire.rate, got.rate)
	assert.Equal(t, fire.lifetime, got.lifetime)
	assert.Equal(t, fire.gravity, got.gravity)
	assert.Equal(t, fire.end, got.end)
	assert.True(t, got.local)
	assert.Equal(t, uint64(3), got.seed)
	assert.Equal(t, geom.V2(4, 4), got.texture.Size())
	fire.Emit(3)
	got.Emit(3)
	assert.Equal(t, fire.live, got.live, "the same seed, the same particles")
}
