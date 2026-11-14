package main

import (
	"fmt"

	"github.com/lantervale/ember/ui"
)

// Health takes damage from one button and heals with the other, their
// callbacks, and shows the result in the orb and the label.
type Health struct {
	orb   *Orb
	label *ui.Node
}

func (h *Health) Hurt() { h.change(-15) }

func (h *Health) Heal() { h.change(10) }

func (h *Health) change(by float32) {
	h.orb.SetHealth(h.orb.Health() + by)
	h.label.SetText(fmt.Sprintf("Health: %.0f", h.orb.Health()))
}
