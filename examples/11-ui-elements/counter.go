package main

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
)

// Counter counts the clicks on one button, and goes back to zero on the
// other: their callbacks.
type Counter struct {
	label *ui.Node
	count int
}

func (c *Counter) Click() { c.count++; c.show() }

func (c *Counter) Reset() { c.count = 0; c.show() }

func (c *Counter) show() { c.label.SetText(fmt.Sprintf("Clicks: %d", c.count)) }
