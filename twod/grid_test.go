package twod //nolint:testpackage // white-box: shares twod's test helpers, reaches unexported state

import (
	"math/rand/v2"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/draw"
	"github.com/lantervale/ember/internal/slot"
	"github.com/lantervale/ember/internal/twodgrid"
	"github.com/lantervale/ember/internal/wm"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/world"
	"github.com/stretchr/testify/require"
)

// busy is a wide scene that moves, grows, adds and takes entities each
// fixed step, from one seed: still sprites over a big world, a y-sorted
// crowd walking, a parallax layer, a fixed HUD, a minimap camera into a
// texture and a zoomed camera following a walker.
type busy struct {
	rand    *rand.Rand
	floor   *Layer
	crowd   *Layer
	far     *Layer
	walkers []*Sprite
	cam     *Camera
	step    int
}

func (*busy) Phase() world.Phase { return world.FixedUpdate }

func (b *busy) Update(f world.Frame) {
	b.step++
	r := b.rand
	for _, s := range b.walkers {
		s.Translate(geom.V2(r.Float32()*16-8, r.Float32()*16-8))
	}
	lead := b.walkers[0]
	lead.Translate(geom.V2(9, 3))
	switch b.step % 40 {
	case 5:
		s := NewSprite(atlas(12, 30), At(lead.Position().X+r.Float32()*400, lead.Position().Y))
		b.crowd.Add(s)
		b.walkers = append(b.walkers, s)
	case 10:
		victim := b.walkers[len(b.walkers)-1]
		f.World.RemoveEntity(victim)
		b.walkers = b.walkers[:len(b.walkers)-1]
	case 15:
		b.walkers[3].SetSize(geom.V2(600, 40))
	case 20:
		b.walkers[4].Add(NewSprite(atlas(4, 4), At(0, -20)))
	case 25:
		b.walkers[5].SetPosition(lead.Position().Add(geom.V2(100, 0)))
		b.walkers[5].Snap()
	case 30:
		s := b.walkers[6]
		s.GetSceneNode().Reparent(b.far.GetSceneNode())
		b.walkers = append(b.walkers[:6], b.walkers[7:]...)
	}
	b.cam.SetZoom(1 + 0.3*geom.Sin(float32(b.step)/30))
}

func busyScene() (*world.World, *platform.Window) {
	w := world.New().AddPlugins(wm.Plugin{}, Plugin{})
	win := platform.NewWindow(platform.Size(800, 600))
	w.AddEntities(win)
	r := rand.New(rand.NewPCG(7, 7))
	floor, crowd := NewLayer(0), NewLayer(2, SortByY())
	far := NewLayer(-1, Parallax(0.5))
	hud := NewLayer(9, Fixed())
	for range 3000 {
		floor.Add(
			NewSprite(
				atlas(8+r.IntN(40), 8+r.IntN(40)),
				At(r.Float32()*8000-4000, r.Float32()*8000-4000),
				Rotation(r.Float32()*3),
				Scale(0.5+r.Float32(), 0.5+r.Float32()),
			),
		)
	}
	floor.Add(NewSprite(atlas(1200, 30), At(0, 0)))
	for range 400 {
		far.Add(NewSprite(atlas(20, 20), At(r.Float32()*6000-3000, r.Float32()*6000-3000)))
	}
	b := &busy{rand: rand.New(rand.NewPCG(9, 9)), floor: floor, crowd: crowd, far: far}
	for range 300 {
		s := NewSprite(atlas(12, 30), At(r.Float32()*3000-1500, r.Float32()*3000-1500), Pivot(0.5, 1))
		crowd.Add(s)
		b.walkers = append(b.walkers, s)
	}
	for range 10 {
		hud.Add(NewSprite(atlas(10, 10), At(r.Float32()*800, r.Float32()*600)))
	}
	b.cam = NewCamera(Follow(b.walkers[0]), Smoothing(6))
	minimap := NewRenderTexture(128, 128)
	win.Add(floor, crowd, far, hud, b.cam, NewCamera(Into(minimap), Follow(b.walkers[0]), Zoom(0.1), Layers(crowd)))
	w.AddEntities(floor, crowd, far, hud)
	w.AddSystems(b)
	return w, win
}

func TestTheGridDrawsExactlyWhatTheWholeWalkDraws(t *testing.T) {
	defer func() { twodgrid.Off = false }()
	twodgrid.Off = true
	plain, plainWin := busyScene()
	twodgrid.Off = false
	fast, fastWin := busyScene()
	for frame := range 240 {
		dt := float32(1) / 45
		twodgrid.Off = true
		plain.Frame(dt)
		twodgrid.Off = false
		fast.Frame(dt)
		want, got := draw.FrontFor(plain, plainWin), draw.FrontFor(fast, fastWin)
		require.NotEmpty(t, want.Unindexed())
		require.Equal(t, want.Unindexed(), got.Unindexed(), "frame %d: the window's list", frame)
		require.Equal(t, want.Batches, got.Batches, "frame %d: its batches", frame)
		wp, gp := draw.PassesFor(plain, plainWin), draw.PassesFor(fast, fastWin)
		require.Len(t, gp, len(wp))
		for i := range wp {
			require.Equal(t, wp[i].List.Unindexed(), gp[i].List.Unindexed(), "frame %d: pass %d", frame, i)
		}
	}
	for _, e := range fast.Entities() {
		if l, ok := e.(*Layer); ok && l.number == 0 {
			require.NotNil(t, l.grid, "the floor files its sprites")
			require.Less(t, len(l.grid.seen), 3001/10, "and visits a tenth of them at most")
		}
	}
}

// TestGatherAndCollectAllocateNothingOnceWarm runs the busy scene a while,
// then counts what twod's gather and collect allocate in a frame.
func TestGatherAndCollectAllocateNothingOnceWarm(t *testing.T) {
	w, _ := busyScene()
	w.RunFrames(30)
	var sys *System
	for _, s := range w.Systems() {
		if v, ok := s.(*System); ok {
			sys = v
		}
	}
	require.NotNil(t, sys)
	pu := slot.Get[windows](w)
	f := world.Frame{DT: 1.0 / 60, World: w, Time: w.Time()}
	run := func() {
		sc := sys.gather(f)
		sc.frame(f, pu)
		sc.prepare(f, pu)
		sys.collectAll(f, pu, sc)
	}
	run()
	require.Zero(t, testing.AllocsPerRun(20, run))
}
