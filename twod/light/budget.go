package light

import (
	"slices"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/internal/draw"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
)

// Priority ranks a light against the others of its view when they are
// more than the camera's Budget: higher ones are kept first, then the
// nearest the middle of the view. 0 by default.
func Priority(n int) Option { return func(l *Light) { l.priority = n } }

func (l *Light) Priority() int     { return l.priority }
func (l *Light) SetPriority(n int) { l.priority = n }

// Budget is how many point, cone and shape lights a camera's view draws
// at most, an option of the camera; 0, the default, draws every one that
// reaches it. Past it, the lights of lowest Priority, then farthest from
// the middle of the view, fade out over BudgetFade, and fade back in when
// they make it again. Directional and Static lights count against none.
func Budget(n int) twod.Option {
	return func(e twod.Entity) { state(e).Budget = max(n, 0) }
}

// BudgetOf is cam's Budget, 0 for none; SetBudget changes it.
func BudgetOf(cam *twod.Camera) int { return state(cam).Budget }

func SetBudget(cam *twod.Camera, n int) { Budget(n)(cam) }

// BudgetFade is how long, in seconds, a light takes to fade out of a
// camera's view as it falls past the Budget, or back in: 0.3 by default,
// 0 for at once.
func BudgetFade(seconds float32) twod.Option {
	return func(e twod.Entity) { s := state(e); s.BudgetFade, s.HasBudgetFade = max(seconds, 0), true }
}

// BudgetFadeOf is cam's BudgetFade; SetBudgetFade changes it.
func BudgetFadeOf(cam *twod.Camera) float32 {
	if s := state(cam); s.HasBudgetFade {
		return s.BudgetFade
	}
	return defaultBudgetFade
}

func SetBudgetFade(cam *twod.Camera, seconds float32) { BudgetFade(seconds)(cam) }

const defaultBudgetFade = 0.3

// kept favours a light already kept against one newly in the running:
// its distance counts this much less, so two lights about as near never
// swap back and forth.
const kept = 0.9

// candidate is a light reaching a window this frame, as the window draws
// it.
type candidate struct {
	light *Light
	drawn draw.Light
	// rank is how near the middle of the view it is, in points.
	rank float32
}

// budget is a window's lights fading in and out of its camera's Budget:
// how far each is in, 0 to 1, and whether it was kept last frame.
type budget struct {
	levels, next map[*Light]float32
	kept         map[*Light]bool
	order        []int
}

// apply appends to l the candidates win's camera draws this frame, the n
// best of them past a budget of n, dt seconds after the last, each faded
// in or out over fade seconds.
func (b *budget) apply(l *draw.Lighting, cands []candidate, win *platform.Window, n int, fade, dt float32) {
	if b.levels == nil {
		b.levels, b.next, b.kept = map[*Light]float32{}, map[*Light]float32{}, map[*Light]bool{}
	}
	middle := win.Size().Scale(0.5)
	b.order = b.order[:0]
	for i := range cands {
		c := &cands[i]
		d := c.drawn.At.Sub(middle).Len()
		c.rank = max(d-c.drawn.Radius, 0) + d*1e-3
		if c.drawn.Kind == draw.ShapeLight {
			r := c.drawn.ShapeBounds()
			centre := geom.V2(r.X+r.W/2, r.Y+r.H/2)
			d = centre.Sub(middle).Len()
			c.rank = max(d-c.drawn.Radius-max(r.W, r.H)/2, 0) + d*1e-3
		}
		if b.kept[c.light] {
			c.rank *= kept
		}
		b.order = append(b.order, i)
	}
	over := n > 0 && len(cands) > n
	if over {
		slices.SortStableFunc(b.order, func(i, j int) int {
			a, c := cands[i], cands[j]
			if a.light.priority != c.light.priority {
				return c.light.priority - a.light.priority
			}
			switch {
			case a.rank < c.rank:
				return -1
			case a.rank > c.rank:
				return 1
			}
			return 0
		})
	}
	clear(b.kept)
	clear(b.next)
	step := float32(1)
	if fade > 0 {
		step = dt / fade
	}
	l.Budget = n
	for k, i := range b.order {
		c := cands[i]
		in := !over || k < n
		level, seen := b.levels[c.light]
		switch {
		case !seen && in:
			level = 1
		case !seen:
			level = 0
		case in:
			level = min(level+step, 1)
		default:
			level = max(level-step, 0)
		}
		if in {
			b.kept[c.light] = true
		}
		b.next[c.light] = level
		if level <= 0 {
			continue
		}
		if level < 1 {
			l.Fading++
			s := level * level * (3 - 2*level)
			c.drawn.Color = c.drawn.Color.Scale(s)
		}
		l.Lights = append(l.Lights, c.drawn)
	}
	b.levels, b.next = b.next, b.levels
}
