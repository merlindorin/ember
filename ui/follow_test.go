package ui_test

import (
	"image"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// followPage is a window showing a sprite, a camera and a Document whose
// tag follows the sprite, offset 10px up.
type followPage struct {
	w      *world.World
	win    *platform.Window
	doc    *ui.Document
	sprite *twod.Sprite
	cam    *twod.Camera
	tag    *ui.Node
}

func tagOffset() geom.Vec2 { return geom.V2(0, -10) }

func newFollowPage(t *testing.T, opts ...ui.DocumentOption) *followPage {
	t.Helper()
	p := &followPage{win: platform.NewWindow(platform.Size(400, 300)), doc: ui.NewDocument(opts...)}
	p.sprite = twod.NewSprite(twod.NewTexture(image.NewNRGBA(image.Rect(0, 0, 8, 8))), twod.At(100, 100))
	p.cam = twod.NewCamera()
	p.tag = ui.Span(ui.Style("display: inline-block; width: 40px; height: 12px"), ui.Text("x"),
		ui.Follow(p.sprite, tagOffset()))
	p.doc.Body().Append(p.tag)
	p.win.Add(p.sprite, p.cam, p.doc)
	p.w = ember.New().AddEntities(p.win)
	p.w.RunFrames(3)
	return p
}

// bottomCentre is the tag's box's bottom centre in window points.
func (p *followPage) bottomCentre() geom.Vec2 {
	r, z := p.tag.Box(), p.doc.Zoom()
	return geom.V2((r.X+r.W/2)*z, (r.Y+r.H)*z)
}

func (p *followPage) assertOn(t *testing.T, why string) {
	t.Helper()
	at, ok := draw.ToScreen(p.w, p.sprite)
	require.True(t, ok, why)
	want := at.Add(tagOffset().Scale(p.doc.Zoom()))
	got := p.bottomCentre()
	assert.InDelta(t, want.X, got.X, 0.5, why)
	assert.InDelta(t, want.Y, got.Y, 0.5, why)
}

func TestATagFollowsAMovingSprite(t *testing.T) {
	p := newFollowPage(t)
	p.assertOn(t, "placed")
	for i := range 3 {
		p.sprite.SetPosition(geom.V2(120+float32(i)*30, 90+float32(i)*20))
		p.w.RunFrames(1)
		p.assertOn(t, "moved")
	}
}

func TestATagFollowsWhileTheCameraFollowsSomethingElse(t *testing.T) {
	p := newFollowPage(t)
	hero := twod.NewText("@", twod.At(0, 0))
	p.win.Add(hero)
	p.cam.SetFollow(hero)
	seen := map[geom.Vec2]bool{}
	for i := range 4 {
		hero.SetPosition(geom.V2(float32(i)*25, float32(i)*10))
		p.w.RunFrames(1)
		p.assertOn(t, "the camera moved")
		at, _ := draw.ToScreen(p.w, p.sprite)
		seen[at] = true
	}
	assert.Len(t, seen, 4, "the still sprite moved on screen with the camera")
}

func TestATagHidesOffScreenAndShowsBackOn(t *testing.T) {
	p := newFollowPage(t)
	p.sprite.SetPosition(geom.V2(5000, 5000))
	p.w.RunFrames(2)
	assert.Contains(t, p.tag.Style(), "visibility: hidden")
	p.sprite.SetPosition(geom.V2(150, 150))
	p.w.RunFrames(2)
	assert.NotContains(t, p.tag.Style(), "visibility")
	p.assertOn(t, "back on screen")
}

func TestZoomHalvesTheTagsPx(t *testing.T) {
	p := newFollowPage(t, ui.Zoom(2))
	p.assertOn(t, "zoomed")
	at, _ := draw.ToScreen(p.w, p.sprite)
	assert.InDelta(t, at.Y/2+tagOffset().Y, p.tag.Box().Y+p.tag.Box().H, 0.5, "in CSS px, half the points")
}

func TestAStillSceneWithAFollowerReplays(t *testing.T) {
	p := newFollowPage(t)
	before := counts(p.w)
	p.w.RunFrames(10)
	after := counts(p.w)
	assert.Equal(t, 10, after["ui replays"]-before["ui replays"])
	for name, v := range after {
		if name != "ui replays" {
			assert.Equal(t, before[name], v, "no %s", name)
		}
	}
}

func TestSetFollowNilLeavesTheTagWhereItWas(t *testing.T) {
	p := newFollowPage(t)
	was := p.tag.Box()
	ui.SetFollow(p.tag, nil, geom.Vec2{})
	p.sprite.SetPosition(geom.V2(300, 200))
	p.w.RunFrames(2)
	assert.Equal(t, was, p.tag.Box())
}

func counts(w *world.World) map[string]int {
	out := map[string]int{}
	for _, c := range w.Stats().Counted() {
		out[c.Name] = c.N
	}
	return out
}

func TestATagMadeToFollowOnceLaidOutFollowsAtOnce(t *testing.T) {
	p := newFollowPage(t)
	still := ui.Span(ui.Style("display: inline-block; width: 40px; height: 12px"), ui.Text("y"))
	p.doc.Body().Append(still)
	p.w.RunFrames(3)
	p.tag = still
	ui.SetFollow(still, p.sprite, tagOffset())
	p.w.RunFrames(2)
	p.assertOn(t, "made to follow on a still page")
}
