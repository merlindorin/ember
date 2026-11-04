package main

import (
	"fmt"

	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// Hit hurts the walker under a left click by a quarter of its health,
// which its bar shows at once, and counts the hits in the score.
type Hit struct {
	walkers []*walker
	mouse   *input.State
	score   *ui.Node
	hits    int
}

func (s *Hit) Update(world.Frame) {
	if !s.mouse.Pressed(input.MouseLeft) {
		return
	}
	at := s.mouse.Mouse()
	for _, w := range s.walkers {
		d := at.Sub(w.sprite.Position())
		if abs(d.X) > 16 || abs(d.Y) > 16 || w.health <= 0 {
			continue
		}
		w.health = max(0, w.health-0.25)
		kit.SetProgress(w.bar, w.health)
		s.hits++
		s.score.SetText(fmt.Sprintf("Hits %d", s.hits))
		return
	}
}
