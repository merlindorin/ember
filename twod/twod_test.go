package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"image"
	"math"
	"strings"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/internal/wm"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func app() (*world.World, *platform.Window, *draw.List) {
	w := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win := platform.NewWindow(platform.Size(800, 600))
	w.AddEntities(win)
	w.RunFrames(1)
	return w, win, draw.FrontFor(w, win)
}

func atlas(w, h int) *Texture { return NewTexture(image.NewRGBA(image.Rect(0, 0, w, h))) }

func corners(vs []draw.Vertex) (geom.Vec2, geom.Vec2) {
	var lo, hi geom.Vec2
	lo, hi = geom.V2(math.MaxFloat32, math.MaxFloat32), geom.V2(-math.MaxFloat32, -math.MaxFloat32)
	for _, v := range vs {
		lo, hi = geom.V2(min(lo.X, v.Pos.X), min(lo.Y, v.Pos.Y)), geom.V2(max(hi.X, v.Pos.X), max(hi.Y, v.Pos.Y))
	}
	return lo, hi
}

func TestSpritesDrawTheirFrameRegionPivotAndFlip(t *testing.T) {
	w, win, c := app()
	sheet := atlas(32, 16)
	s := NewSprite(sheet, At(100, 100), Still(Strip(sheet, Cell(16, 16))[1]))
	win.AppendChild(s)
	w.AddEntities(s)
	w.RunFrames(1)
	require.Len(t, c.Unindexed(), 6)
	lo, hi := corners(c.Unindexed())
	assert.Equal(t, geom.V2(92, 92), lo, "centred on its position, one 16×16 frame")
	assert.Equal(t, geom.V2(108, 108), hi)
	assert.Equal(t, geom.V2(0.5, 0), c.Unindexed()[0].UV, "the second frame")
	assert.Equal(t, geom.V2(1, 1), c.Unindexed()[2].UV)

	s.SetFlip(true, false)
	s.SetStill(Frame{Texture: sheet, Region: geom.Rect{X: 16, W: 16, H: 8}})
	Pivot(0, 0).sprite(s)
	w.RunFrames(1)
	lo, hi = corners(c.Unindexed())
	assert.Equal(t, geom.V2(100, 100), lo, "a top-left pivot")
	assert.Equal(t, geom.V2(116, 108), hi, "the region's size")
	assert.Equal(t, geom.V2(1, 0), c.Unindexed()[0].UV, "flipped region: u runs backwards")
	assert.Equal(t, geom.V2(0.5, 0.5), c.Unindexed()[2].UV)
	assert.Equal(t, geom.Rect{W: 16, H: 8}, s.Rect())
}

func TestTheSharedOptionsAndSetters(t *testing.T) {
	s := NewSprite(nil, At(1, 2), Rotation(0.5), Scale(2, 3), Z(4), Tint(color.Red), Hidden(), SortByY())
	if s.Position() != geom.V2(1, 2) || s.Rotation() != 0.5 || s.Scale() != geom.V2(2, 3) || s.Z() != 4 ||
		s.Tint() != color.Red || !s.Hidden() || !s.SortByY() {
		t.Fatalf("options: %+v", s.Item)
	}
	s.Translate(geom.V2(1, 1))
	s.SetHidden(false)
	if s.Position() != geom.V2(2, 3) || s.Hidden() {
		t.Fatal("setters")
	}
	d := NewSprite(nil)
	if d.Scale() != geom.V2(1, 1) || d.Tint() != color.White || d.Hidden() || d.Name() != "sprite" {
		t.Fatal("defaults")
	}
}

func TestChildrenMoveTintAndHideWithTheirParent(t *testing.T) {
	w, win, c := app()
	parent := NewSprite(atlas(10, 10), At(100, 0), Scale(2, 2), Tint(color.RGBA(1, 0.5, 1, 1)), Pivot(0, 0))
	child := NewSprite(atlas(10, 10), At(10, 0), Tint(color.RGBA(0.5, 1, 1, 1)), Pivot(0, 0))
	parent.Add(child)
	win.AppendChild(parent)
	w.AddEntities(parent)
	w.RunFrames(1)
	require.Len(t, c.Unindexed(), 12)
	assert.Equal(t, geom.V2(120, 0), c.Unindexed()[6].Pos, "the child's position is scaled by its parent's")
	assert.Equal(t, color.RGBA(0.5, 0.5, 1, 1), c.Unindexed()[6].Color, "tints multiply")
	parent.SetHidden(true)
	w.RunFrames(1)
	assert.Empty(t, c.Unindexed(), "hiding the parent hides its children")
}

func TestLayersZAndYDecideTheOrder(t *testing.T) {
	w, win, c := app()
	tex := atlas(2, 2)
	back := NewLayer(-1)
	front := NewLayer(1)
	world2 := NewLayer(0, SortByY())
	low := NewSprite(tex, At(1, 50))
	high := NewSprite(tex, At(2, 10))
	top := NewSprite(tex, At(3, 0), Z(5))
	world2.Add(low, high, top)
	back.Add(NewSprite(tex, At(4, 0)))
	front.Add(NewSprite(tex, At(5, 0)))
	win.AppendChild(front)
	win.AppendChild(world2)
	win.AppendChild(back)
	w.AddEntities(front, world2, back)
	w.RunFrames(1)
	var order []float32
	for i := 0; i < len(c.Unindexed()); i += 6 {
		lo, _ := corners(c.Unindexed()[i : i+6])
		order = append(order, lo.X+1)
	}
	assert.Equal(t, []float32{2, 1, 3, 5}, order, "layers by number; by y within a sorted parent; z on top")
	behind := draw.BackFor(w, win)
	require.Len(t, behind.Unindexed(), 6, "a negative layer draws behind the 3D scene")
	lo, _ := corners(behind.Unindexed())
	assert.Equal(t, float32(4), lo.X+1)
}

func TestAWindowShowsItsHighestPriorityCamera(t *testing.T) {
	w, win, c := app()
	hero := NewSprite(atlas(10, 10), At(1000, 500))
	hud := NewLayer(10, Fixed())
	hud.Add(NewSprite(atlas(10, 10), At(5, 5), Pivot(0, 0)))
	far := NewLayer(-5, Parallax(0.5))
	far.Add(NewSprite(atlas(10, 10), At(600, 300)))
	low := NewCamera(Follow(hero), Priority(-1), Zoom(4))
	cam := NewCamera(Follow(hero), Zoom(2), In(win))
	for _, e := range []Entity{hero, hud, far, low, cam} {
		win.AppendChild(e)
		w.AddEntities(e)
	}
	w.RunFrames(1)
	assert.Equal(t, geom.V2(1000, 500), cam.Center())
	require.Len(t, c.Unindexed(), 12)
	farLo, _ := corners(draw.BackFor(w, win).Unindexed())
	heroLo, heroHi := corners(c.Unindexed()[0:6])
	hudLo, _ := corners(c.Unindexed()[6:12])
	assert.Equal(t, geom.V2(390, 290), heroLo, "the followed sprite sits in the middle, twice as big")
	assert.Equal(t, geom.V2(410, 310), heroHi)
	assert.Equal(t, geom.V2(590, 390), farLo, "the far layer moves at half the camera's speed, so it lags behind")
	assert.Equal(t, geom.V2(5, 5), hudLo, "a fixed layer stays put")

	cam.SetZoom(1)
	Bounds(geom.Rect{W: 1200, H: 700}).camera(cam)
	w.RunFrames(1)
	assert.Equal(t, geom.V2(800, 400), cam.Center(), "the view stays inside its bounds")
}

func TestCamerasSmoothAndSnap(t *testing.T) {
	w, win, c := app()
	target := NewSprite(atlas(4, 4), At(0, 0), Pivot(0, 0))
	cam := NewCamera(Follow(target), Smoothing(10), SnapToPixels())
	win.AppendChild(target)
	win.AppendChild(cam)
	w.AddEntities(target, cam)
	w.RunFrames(1)
	target.SetPosition(geom.V2(100.3, 0))
	w.Tick(0.05)
	assert.InDelta(t, 50.15, cam.Center().X, 1e-3, "half way in 0.05 s at speed 10")
	lo, _ := corners(c.Unindexed())
	assert.Equal(t, lo.X, float32(math.Round(float64(lo.X))), "snapped to a whole pixel")
}

func TestACameraShowsInTheWindowItIsUnder(t *testing.T) {
	w, win, _ := app()
	tool := platform.NewWindow(platform.Size(300, 200))
	sprite := NewSprite(atlas(10, 10), At(1000, 500))
	group := NewLayer(0)
	group.Add(NewCamera(Follow(sprite)))
	tool.AppendChild(sprite)
	tool.AppendChild(group)
	w.AddEntities(tool)
	w.RunFrames(1)
	at, ok := toScreen(w, sprite)
	require.True(t, ok)
	assert.Equal(t, geom.V2(150, 100), at, "the tool window's camera, though it is inside a layer")
	assert.Empty(t, draw.FrontFor(w, win).Unindexed(), "nothing in the primary window")
}

func TestEntitiesShowInTheWindowTheyAreAChildOf(t *testing.T) {
	w, primary, c := app()
	tool := platform.NewWindow(platform.Size(300, 300))
	w.AddEntities(tool)
	in := NewSprite(atlas(4, 4))
	tool.AppendChild(in)
	w.AddEntities(in, NewSprite(atlas(4, 4)))
	w.RunFrames(1)
	assert.Len(t, c.Unindexed(), 6, "one sprite in the primary window")
	assert.Len(t, draw.FrontFor(w, tool).Unindexed(), 6, "one in the tool window")
	_ = primary
}

type ship struct {
	Item
	drawn int
}

func (s *ship) Draw(c *Canvas) {
	s.drawn++
	c.Rect(geom.Rect{W: 10, H: 4}, color.White)
}

func TestAGamesOwnEntityEmbedsTheBase(t *testing.T) {
	w, win, c := app()
	s := Build(&ship{Item: Base()}, "ship", At(30, 40))
	win.AppendChild(s)
	w.AddEntities(s)
	w.RunFrames(1)
	assert.Equal(t, 1, s.drawn)
	lo, _ := corners(c.Unindexed())
	assert.Equal(t, geom.V2(30, 40), lo, "drawn in its own space")
	assert.Equal(t, "ship", s.Name())
}

func TestTextWrapsAndNinePatchesStretch(t *testing.T) {
	text := NewText("one two three four", FontSize(20), Wrap(80))
	assert.Greater(t, len(text.Lines()), 1, "wrapped")
	assert.Equal(t, []string{"one two three four"}, NewText("one two three four").Lines())
	assert.Equal(t, []string{"a", "b"}, NewText("a\nb").Lines())
	assert.Equal(t, float32(len(text.Lines()))*25, text.Measure().Y, "its size is its lines")
}

func TestAtPlacesTheCentreOfSizedEntitiesUnlessPivotSaysOtherwise(t *testing.T) {
	drawn := func(e Entity) (geom.Vec2, geom.Vec2) {
		w, win, c := app()
		win.AppendChild(e)
		w.AddEntities(e)
		w.RunFrames(1)
		return corners(c.Unindexed())
	}
	lo, hi := drawn(NewNinePatch(atlas(12, 12), Margins(4, 4, 4, 4), Size(100, 40), At(200, 100)))
	assert.Equal(t, geom.V2(150, 80), lo, "a nine-patch is centred on its position")
	assert.Equal(t, geom.V2(250, 120), hi)
	lo, _ = drawn(NewNinePatch(atlas(12, 12), Margins(4, 4, 4, 4), Size(100, 40), At(200, 100), Pivot(0, 0)))
	assert.Equal(t, geom.V2(200, 100), lo, "Pivot(0, 0) places its top-left corner")
	lo, hi = drawn(NewText("centre", At(200, 100)))
	assert.InDelta(t, 200, (lo.X+hi.X)/2, 2, "a text is centred too")
	lo, _ = drawn(NewText("corner", At(200, 100), Pivot(0, 0)))
	assert.GreaterOrEqual(t, lo.X, float32(200))
	lo, _ = drawn(NewSprite(atlas(10, 10), At(200, 100), Pivot(0, 0)))
	assert.Equal(t, geom.V2(200, 100), lo, "Pivot works on sprites the same way")
}

type badge struct {
	Item
	drawn int
	ended bool
}

func (b *badge) BeginDraw(c *Canvas) func() {
	c.SetSize(geom.V2(4, 4))
	return func() { b.ended = true }
}

func (b *badge) Draw(c *Canvas) {
	b.drawn++
	c.Rect(geom.Rect{W: c.Size().X, H: c.Size().Y}, color.White)
}

func (b *badge) Expose() (string, any) { return "meter", 7 }

func TestOverlaysDrawFixedAboveEveryLayerPreparedAndExposed(t *testing.T) {
	w, win, c := app()
	reg := automation.NewRegistry()
	reg.SetRecording(true)
	slot.Set(w, reg)
	group := NewSprite(nil, At(1, 1))
	group.SetName("hud")
	b := Build(&badge{Item: Base()}, "badge", Overlay(), At(10, 10))
	group.Add(b)
	Overlay()(group)
	top := NewLayer(100)
	top.Add(NewSprite(atlas(2, 2), At(1000, 1000)))
	cam := NewCamera(At(1000, 1000))
	for _, e := range []Entity{group, top, cam} {
		win.AppendChild(e)
		w.AddEntities(e)
	}
	w.RunFrames(1)
	require.Equal(t, 1, b.drawn)
	assert.True(t, b.ended, "what BeginDraw set up is undone after Draw")
	require.Len(t, c.Unindexed(), 12)
	lo, hi := corners(c.Unindexed()[6:])
	assert.Equal(t, geom.V2(11, 11), lo, "drawn last, and the camera does not move it")
	assert.Equal(t, geom.V2(15, 15), hi, "at the size BeginDraw set")
	reg.EndFrame(1)
	_, els := reg.Snapshot("", "")
	require.Len(t, els, 1)
	assert.Equal(t, "hud/badge", els[0].ID, "named after it, scoped by the nodes above")
	assert.Equal(t, 7, els[0].Value)
	assert.Equal(t, geom.Rect{X: 11, Y: 11, W: 4, H: 4}, els[0].Bounds)
}

func TestToScreenIsWhereAnEntityShowedLastFrame(t *testing.T) {
	w, win, _ := app()
	hero := NewSprite(atlas(4, 4), At(1000, 500))
	far := NewLayer(-1, Parallax(0.5))
	tree := NewSprite(atlas(4, 4), At(1000, 500))
	far.Add(tree)
	hud := NewLayer(9, Fixed())
	badge := NewSprite(atlas(4, 4), At(20, 30))
	hud.Add(badge)
	cam := NewCamera(Follow(hero), Zoom(2))
	for _, e := range []Entity{hero, far, hud, cam} {
		win.AppendChild(e)
		w.AddEntities(e)
	}
	_, ok := toScreen(world.New(), hero)
	assert.False(t, ok, "a World that never drew 2D")
	w.RunFrames(1)
	at, ok := toScreen(w, hero)
	assert.True(t, ok)
	assert.Equal(t, geom.V2(400, 300), at, "the followed sprite is in the middle")
	at, _ = toScreen(w, tree)
	assert.Equal(t, geom.V2(1400, 800), at, "through its layer's parallax")
	at, _ = toScreen(w, badge)
	assert.Equal(t, geom.V2(20, 30), at, "a fixed layer ignores the camera")
}

func TestALayerClipsItsItemsToItsRect(t *testing.T) {
	w, win, c := app()
	window := NewLayer(1, ClipTo(geom.Rect{W: 50, H: 30}), At(100, 100))
	window.Add(NewSprite(atlas(10, 10), At(25, 15)), NewSprite(atlas(10, 10), At(200, 200)))
	win.AppendChild(window)
	w.AddEntities(window)
	w.RunFrames(1)
	require.NotEmpty(t, c.Batches)
	for _, b := range c.Batches {
		if !b.Clipped || b.Clip != (geom.Rect{X: 100, Y: 100, W: 50, H: 30}) {
			t.Fatalf("drawn inside the layer's clip, in its space: %+v", b)
		}
	}
}

// roundedPanel draws a square past a rounded clip of its own space.
type roundedPanel struct{ Item }

func (p *roundedPanel) Draw(c *Canvas) {
	c.PushClipRounded(geom.Rect{W: 40, H: 20}, [4]float32{6, 6, 2, 2})
	c.Rect(geom.Rect{W: 100, H: 100}, color.White)
	c.PopClip()
}

func TestACanvasRoundedClipIsPlacedInTheWindowWithItsEntity(t *testing.T) {
	w, win, c := app()
	panel := Build(&roundedPanel{Item: Base()}, "panel", At(100, 50))
	win.AppendChild(panel)
	w.AddEntities(panel)
	w.RunFrames(1)
	require.NotEmpty(t, c.Batches)
	b := c.Batches[len(c.Batches)-1]
	want := draw.Rounded{Rect: geom.Rect{X: 100, Y: 50, W: 40, H: 20}, Radii: [4]float32{6, 6, 2, 2}}
	if !b.Clipped || b.Round != want {
		t.Fatalf("the rounded clip is in the window, where the entity is: %+v", b)
	}
}

func TestAddTakesAnyChildrenAndReturnsTheParent(t *testing.T) {
	w, win, c := app()
	body := &struct{ *world.BasicEntity }{world.NewBasic()}
	hero := NewSprite(atlas(8, 8))
	layer := NewLayer(1).Add(hero.Add(body), NewSprite(atlas(8, 8)))
	assert.IsType(t, &Layer{}, layer, "the ready-made entities return themselves")
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	assert.True(t, w.Has(body), "a plain entity is a child too")
	assert.Equal(t, hero.BasicEntity, body.Parent())
	assert.Equal(t, layer.GetSceneNode(), hero.GetSceneNode().ParentNode(), "and 2D children join the node tree")
	assert.Len(t, c.Unindexed(), 12)
}

func TestASpriteFitsItsPictureToItsSize(t *testing.T) {
	w, win, c := app()
	s := NewSprite(atlas(32, 16), At(100, 100), Size(64, 64))
	win.Add(s)
	w.AddEntities(s)
	w.RunFrames(1)
	lo, hi := corners(c.Unindexed())
	assert.Equal(t, geom.V2(68, 68), lo, "centred, 64×64 points")
	assert.Equal(t, geom.V2(132, 132), hi)
	assert.Equal(t, geom.V2(1, 1), c.Unindexed()[2].UV, "the whole picture")
	s.SetSize(geom.Vec2{})
	assert.Equal(t, geom.V2(32, 16), s.Size(), "its own size again")
}

func TestARemovedEntityStopsDrawing(t *testing.T) {
	w, win, c := app()
	gone := NewSprite(atlas(8, 8))
	layer := NewLayer(0).Add(gone, NewSprite(atlas(8, 8)))
	win.Add(layer)
	w.AddEntities(layer)
	w.RunFrames(1)
	w.RemoveEntity(gone)
	w.RunFrames(1)
	assert.Len(t, c.Unindexed(), 6, "only the sprite left")
	assert.Len(t, layer.GetSceneNode().ChildNodes(), 1, "it left the layer's tree too")
	assert.Nil(t, gone.Parent())
}

func TestNameNamesAnEntityAndAlignIsSavedAsWords(t *testing.T) {
	assert.Equal(t, "hero", NewSprite(nil, Name("hero")).GetSceneNode().Name())
	b, err := AlignCenter.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "center", string(b))
	var a Align
	require.NoError(t, a.UnmarshalText([]byte("right")))
	assert.Equal(t, AlignRight, a)
}

func TestTheSystemRecordsItsCollectAndDrawTimes(t *testing.T) {
	w, win, _ := app()
	s := NewSprite(atlas(8, 8), At(10, 10))
	win.AppendChild(s)
	w.AddEntities(s)
	w.RunFrames(2)
	names := map[string]bool{}
	for _, r := range w.Stats().Recorded() {
		names[r.Name] = true
	}
	assert.True(t, names["twod collect"] && names["twod draw"], "%v", names)
}

func TestTheF3OverlayListsTheCounters(t *testing.T) {
	w, _, _ := app()
	w.RunFrames(1)
	w.Stats().Count("ui replays")
	w.Stats().Count("ui replays")
	w.Stats().Count("ui repaints: input")
	text := strings.Join(debugCosts(nil, nil, w.Stats().Counted()), "\n")
	assert.Contains(t, text, "counted\n")
	assert.Regexp(t, `ui replays +2`, text)
	assert.Regexp(t, `ui repaints: input +1`, text)
	assert.NotContains(t, strings.Join(debugCosts(nil, nil, nil), "\n"), "counted", "no section without counters")
}

func TestTheF3OverlayListsTheRecordedCosts(t *testing.T) {
	w, win, _ := app()
	s := NewSprite(atlas(8, 8), At(10, 10))
	win.AppendChild(s)
	w.AddEntities(s)
	w.RunFrames(2)
	lines := debugCosts(w.Stats().Slowest(8), w.Stats().Recorded(), w.Stats().Counted())
	text := strings.Join(lines, "\n")
	assert.Contains(t, text, "twod", "the slowest systems")
	assert.Contains(t, text, "recorded\n")
	assert.Contains(t, text, "twod collect")
	assert.Contains(t, text, "twod draw")
	head := debugHead(w.Stats())
	assert.Len(t, head, 6)
	assert.True(t, strings.HasPrefix(head[1], "work "), "the work apart from the display's wait: %q", head[1])
	assert.Contains(t, head[1], "fps possible")
}

func TestTheProfilerShowsByF3OrByCall(t *testing.T) {
	w, _, _ := app()
	assert.False(t, ProfilerShown(w))
	ShowProfiler(w, true)
	w.RunFrames(1)
	assert.True(t, ProfilerShown(w), "shown by a call, as a debug panel does")
	wm.Input(w).SetKey(input.KeyF3, true)
	w.RunFrames(2)
	assert.False(t, ProfilerShown(w), "and F3 hides it again: one switch")
}
