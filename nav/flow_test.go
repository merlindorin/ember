package nav //nolint:testpackage // white-box: the field's build counters and the level helpers of nav_test.go

import (
	"math/rand/v2"
	"testing"

	"github.com/merlindorin/ember/collide"
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/world"
)

// SolidIn makes a level a collide.Solid: the rectangles of its solid cells.
func (l *level) SolidIn(area geom.Rect) []geom.Rect {
	var out []geom.Rect
	x0, y0 := int(geom.Floor(area.X/10)), int(geom.Floor(area.Y/10))
	x1, y1 := int(geom.Floor((area.X+area.W)/10)), int(geom.Floor((area.Y+area.H)/10))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if y >= 0 && x >= 0 && y < len(l.cells) && x < len(l.cells[0]) && l.cells[y][x] == '#' {
				out = append(out, geom.Rect{X: float32(x) * 10, Y: float32(y) * 10, W: 10, H: 10})
			}
		}
	}
	return out
}

type walker struct {
	*world.BasicEntity
	at geom.Vec2
}

func (w *walker) Position() geom.Vec2     { return w.at }
func (w *walker) SetPosition(p geom.Vec2) { w.at = p }

// walkers puts a body on each point, in a space where l's walls are solid
// and the walkers stop each other.
func walkers(l *level, points ...geom.Vec2) (*collide.Space, []*walker) {
	w := world.New()
	space := collide.NewSpace(collide.Grid(10))
	w.AddSystems(collide.NewTracker(space))
	space.AddSolid(l)
	var out []*walker
	for _, p := range points {
		e := &walker{BasicEntity: world.NewBasic(), at: p}
		e.AppendChild(collide.NewBody(collide.Box(6, 4), collide.Group(1), collide.CollidesWith(1)))
		w.AddEntities(e)
		out = append(out, e)
	}
	return space, out
}

// chase moves each walker a point a step for steps, along the field, else
// straight at the target, as a game's Chase would.
func chase(space *collide.Space, f *FlowField, target geom.Vec2, steps int, ws ...*walker) {
	f.SetTarget(target)
	for range steps {
		for _, e := range ws {
			dir, ok := f.Toward(e.at)
			if !ok {
				dir = target.Sub(e.at).Normalize()
			}
			space.Move(e, dir)
		}
	}
}

func column() []string {
	return []string{
		"..............................",
		"..............................",
		"..............................",
		"..............###.............",
		"..............###.............",
		"..............###.............",
		"..............###.............",
		"..............###.............",
		"..............................",
		"..............................",
		"..............................",
	}
}

func TestAnEnemyBehindAColumnGoesRoundIt(t *testing.T) {
	l := newLevel(column()...)
	space, ws := walkers(l, at(2, 5))
	f := NewFlowField(l)
	hero := at(27, 5)
	chase(space, f, hero, 400, ws...)
	if ws[0].at.Dist(hero) > 6 {
		t.Fatalf("round the column to the hero: %v", ws[0].at)
	}
}

func TestACrowdFlowsRoundBothSidesOfAColumn(t *testing.T) {
	l := newLevel(column()...)
	var points []geom.Vec2
	for y := 2; y <= 8; y++ {
		points = append(points, at(2, y), at(4, y))
	}
	space, ws := walkers(l, points...)
	f := NewFlowField(l)
	above, below := false, false
	f.SetTarget(at(27, 5))
	for range 500 {
		chase(space, f, at(27, 5), 1, ws...)
		for _, w := range ws {
			if w.at.X > 140 && w.at.X < 170 {
				above = above || w.at.Y < 30
				below = below || w.at.Y > 80
			}
		}
	}
	past := 0
	for _, w := range ws {
		if w.at.X > 170 {
			past++
		}
	}
	if past != len(ws) || !above || !below {
		t.Fatalf("all %d past the column, round both sides (above %v, below %v): %d past", len(ws), above, below, past)
	}
}

func TestACorridorBlockedAheadStillQueues(t *testing.T) {
	l := newLevel(
		"##############################",
		"............#.................",
		"##############################",
	)
	space, ws := walkers(l, at(6, 1), at(2, 1))
	f := NewFlowField(l)
	chase(space, f, at(25, 1), 300, ws...)
	if _, ok := f.Toward(ws[0].at); ok {
		t.Fatal("the hero can't be reached past the block")
	}
	if ws[0].at.X < 120-4-0.1 || ws[1].at.X < ws[0].at.X-6-0.1 {
		t.Fatalf("straight on, the front one at the block, the other behind it: %v %v", ws[0].at, ws[1].at)
	}
}

func TestTheFieldIsMadeAgainOnlyWhenItMustBe(t *testing.T) {
	l := newLevel(column()...)
	f := NewFlowField(l)
	f.SetTarget(at(27, 5))
	f.SetTarget(at(27, 5).Add(geom.V2(2, 3)))
	if f.builds != 1 {
		t.Fatalf("the same cell: made once, not %d times", f.builds)
	}
	f.SetTarget(at(26, 5))
	l.set(0, 0, '#')
	f.SetTarget(at(26, 5))
	if f.builds != 3 {
		t.Fatalf("another cell, then a changed grid: %d", f.builds)
	}
}

func TestWhereTheFieldDoesNotLead(t *testing.T) {
	l := newLevel(
		"..........",
		".####.....",
		".#..#.....",
		".####.....",
	)
	f := NewFlowField(l)
	if _, ok := f.Toward(at(0, 0)); ok {
		t.Fatal("no target yet")
	}
	f.SetTarget(at(8, 2))
	for _, p := range []geom.Vec2{at(2, 2), at(1, 1), geom.V2(-5, 5), at(20, 2)} {
		if _, ok := f.Toward(p); ok {
			t.Errorf("%v: walled in, solid, or outside: no way", p)
		}
	}
	if dir, ok := f.Toward(at(0, 3)); !ok || dir.LenSq() < 0.99 {
		t.Errorf("a reachable cell has a unit way: %v %v", dir, ok)
	}
	f.SetTarget(at(1, 1))
	if _, ok := f.Toward(at(0, 0)); ok {
		t.Error("a target in a wall leads nowhere")
	}
}

// endless is an open grid without edges, cut into windows on demand.
type endless struct{ cuts int }

type window struct{ x, y, cols, rows int }

func (w window) Columns() int        { return w.cols }
func (w window) Rows() int           { return w.rows }
func (w window) Solid(x, y int) bool { return x < 0 || y < 0 || x >= w.cols || y >= w.rows }
func (w window) Cell(p geom.Vec2) (int, int, bool) {
	x, y := int(geom.Floor(p.X/10))-w.x, int(geom.Floor(p.Y/10))-w.y
	return x, y, x >= 0 && y >= 0 && x < w.cols && y < w.rows
}
func (w window) Center(x, y int) geom.Vec2 {
	return geom.V2(float32(x+w.x)*10+5, float32(y+w.y)*10+5)
}

func (e *endless) around(p geom.Vec2, cols, rows int) window {
	e.cuts++
	return window{int(geom.Floor(p.X/10)) - cols/2, int(geom.Floor(p.Y/10)) - rows/2, cols, rows}
}

func TestAFieldAroundFollowsItsTargetOverAnEndlessGrid(t *testing.T) {
	var e endless
	f := NewFlowField(nil, Around(e.around, 32, 32), NoDiagonals())
	f.SetTarget(geom.V2(1000, 1000))
	if dir, ok := f.Toward(geom.V2(950, 1000)); !ok || dir.X < 0.9 {
		t.Fatalf("towards the target: %v %v", dir, ok)
	}
	f.SetTarget(geom.V2(1050, 1000))
	if e.cuts != 1 {
		t.Fatalf("within the window's middle half, the same window: %d cuts", e.cuts)
	}
	f.SetTarget(geom.V2(1200, 1000))
	if e.cuts != 2 || f.builds != 3 {
		t.Fatalf("strayed: a new window, made again: %d cuts, %d builds", e.cuts, f.builds)
	}
	if _, ok := f.Toward(geom.V2(1150, 1000)); !ok {
		t.Fatal("the new window covers around the target")
	}
	if f.Diagonals() {
		t.Fatal("NoDiagonals")
	}
}

func BenchmarkFlowField128(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	rows := make([]string, 128)
	for y := range rows {
		row := make([]byte, 128)
		for x := range row {
			row[x] = '.'
			if r.Float32() < 0.2 {
				row[x] = '#'
			}
		}
		rows[y] = string(row)
	}
	l := newLevel(rows...)
	f := NewFlowField(l)
	b.ResetTimer()
	for i := range b.N {
		l.version++
		f.SetTarget(at(64+i%2, 64))
	}
}

func BenchmarkFlowToward(b *testing.B) {
	l := newLevel(column()...)
	f := NewFlowField(l)
	f.SetTarget(at(27, 5))
	b.ResetTimer()
	for i := range b.N {
		f.Toward(geom.V2(float32(i%250), 33))
	}
}

func open(n int) *level {
	rows := make([]string, n)
	for y := range rows {
		row := make([]byte, n)
		for x := range row {
			row[x] = '.'
		}
		rows[y] = string(row)
	}
	return newLevel(rows...)
}

func TestInTheOpenEveryEnemyComesStraightAtTheTarget(t *testing.T) {
	f := NewFlowField(open(64))
	target := at(32, 32)
	f.SetTarget(target)
	for deg := 120.0; deg <= 240; deg += 7 {
		from := target.Add(geom.FromAngle(float32(deg) * geom.Pi / 180).Scale(200))
		dir, ok := f.Toward(from)
		want := target.Sub(from).Normalize()
		if !ok || dir.Sub(want).Len() > 1e-4 {
			t.Errorf("from %v°: %v, want straight at it %v, not one of eight ways", deg, dir, want)
		}
	}
}

func TestBehindAColumnTheWayStillGoesRound(t *testing.T) {
	l := newLevel(column()...)
	f := NewFlowField(l)
	f.SetTarget(at(27, 5))
	if f.shown.seen[f.shown.index(2, 5)] {
		t.Fatal("the column hides the target from straight behind it")
	}
	if !f.shown.seen[f.shown.index(14, 0)] {
		t.Fatal("from above the column's top, it is in sight")
	}
	if dir, ok := f.Toward(at(10, 5)); !ok || geom.Abs(dir.Y) < 0.3 {
		t.Fatalf("behind it, the field's way, bending round: %v", dir)
	}
}

// spread is how far apart walkers are across the hero's way: the standard
// deviation of their y.
func spread(ws []*walker) float32 {
	var sum, sq float32
	for _, w := range ws {
		sum += w.at.Y
	}
	mean := sum / float32(len(ws))
	for _, w := range ws {
		sq += (w.at.Y - mean) * (w.at.Y - mean)
	}
	return geom.Sqrt(sq / float32(len(ws)))
}

func TestChasersOfARunningHeroStaySpreadNotInSingleFile(t *testing.T) {
	l := open(128)
	r := rand.New(rand.NewPCG(3, 4))
	var points []geom.Vec2
	for range 50 {
		points = append(points, geom.V2(100+r.Float32()*200, 300+r.Float32()*600))
	}
	space, ws := walkers(l, points...)
	f := NewFlowField(l)
	hero := geom.V2(500, 640)
	for range 400 {
		hero = hero.Add(geom.V2(1.5, 0))
		f.SetTarget(hero)
		for _, e := range ws {
			dir, ok := f.Toward(e.at)
			if !ok {
				dir = hero.Sub(e.at).Normalize()
			}
			space.Move(e, dir)
		}
	}
	if s := spread(ws); s < 40 {
		t.Fatalf("after 400 steps the chasers stand %.0f apart across his way: a single file (the 8-way field left 10)", s)
	}
}

func TestABackgroundFieldShowsAtTheNextCallTheFieldMadeAtOnce(t *testing.T) {
	l := newLevel(column()...)
	f, now := NewFlowField(l, Background()), NewFlowField(l)
	same := func(when string) {
		t.Helper()
		for y := float32(0); y < 110; y += 3.7 {
			for x := float32(0); x < 300; x += 4.3 {
				p := geom.V2(x, y)
				a, okA := f.Toward(p)
				b, okB := now.Toward(p)
				if a != b || okA != okB {
					t.Fatalf("%s, at %v: %v %v, made at once %v %v", when, p, a, okA, b, okB)
				}
			}
		}
	}
	f.SetTarget(at(27, 5))
	now.SetTarget(at(27, 5))
	same("the first field is made at once")

	f.SetTarget(at(2, 5))
	if f.shown.goal != 5*30+27 {
		t.Fatal("until the next call, the field before leads")
	}
	f.SetTarget(at(2, 5))
	now.SetTarget(at(2, 5))
	if f.builds != 2 {
		t.Fatalf("the next call shows it, made once: %d builds", f.builds)
	}
	same("the next call")

	l.set(10, 5, '#')
	f.SetTarget(at(2, 5))
	f.SetTarget(at(2, 5))
	now.SetTarget(at(2, 5))
	same("a changed grid")
	if !f.Background() || now.Background() {
		t.Fatal("Background")
	}
}

// roomy is a level where some solid cells still leave a body room, at the
// point given.
type roomy struct {
	*level
	rooms map[[2]int]geom.Vec2
}

func (r roomy) Pass(_, _, _ geom.Vec2) bool { return true }

func (r roomy) Room(x, y int, _ geom.Vec2) (geom.Vec2, bool) {
	if p, ok := r.rooms[[2]int{x, y}]; ok {
		return p, true
	}
	if r.Solid(x, y) {
		return geom.Vec2{}, false
	}
	return r.Center(x, y), true
}

func TestABodyRoutesThroughASolidCellItFitsIn(t *testing.T) {
	l := newLevel(
		"..........",
		"#####.####",
		"..........",
	)
	l.set(5, 1, '#')
	gap := at(5, 1).Add(geom.V2(-2, 0))
	grid := roomy{l, map[[2]int]geom.Vec2{{5, 1}: gap}}
	whole := NewFlowField(grid)
	whole.SetTarget(at(5, 0))
	if _, ok := whole.Toward(at(5, 2)); ok {
		t.Fatal("without a body, the wall's only gap is shut")
	}
	f := NewFlowField(grid, Body(8, 8))
	f.SetTarget(at(5, 0))
	dir, ok := f.Toward(at(5, 2))
	if !ok || dir.Y > -0.5 {
		t.Fatalf("with a body that fits, up through the gap: %v %v", dir, ok)
	}
	if got := f.shown.from(5, 1); got != gap {
		t.Fatalf("its way starts where the body fits: %v", got)
	}
	if f.shown.seen[f.shown.index(5, 2)] {
		t.Fatal("a partly solid cell still blocks the line of sight")
	}
	if f.Body() != geom.V2(8, 8) {
		t.Fatal("Body")
	}
}
