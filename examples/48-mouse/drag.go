package main

import (
	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/world"
)

// move is a crate taken from one place to another, kept to undo it.
type move struct {
	crate    *twod.Sprite
	from, to geom.Vec2
}

// Drag lets the left button carry a crate, the wheel zoom the camera, and
// the Back and Forward buttons walk through the moves made.
type Drag struct {
	window *platform.Window
	eye    *twod.Camera
	crates []*twod.Sprite
	carry  platform.Cursor

	held  *twod.Sprite
	grab  geom.Vec2
	from  geom.Vec2
	moves []move
	done  int
}

func (d *Drag) Update(world.Frame) {
	in := d.window.Input()
	at := d.eye.ScreenToWorld(in.Mouse())
	under := d.crateAt(at)

	switch {
	case in.Pressed(input.MouseLeft) && under != nil:
		d.held, d.grab, d.from = under, under.Position().Sub(at), under.Position()
	case in.Down(input.MouseLeft) && d.held != nil:
		d.held.SetPosition(at.Add(d.grab))
	case in.Released(input.MouseLeft) && d.held != nil:
		d.moves = append(d.moves[:d.done], move{d.held, d.from, d.held.Position()})
		d.done = len(d.moves)
		d.held = nil
	}

	if in.Pressed(input.MouseBack) && d.done > 0 {
		d.done--
		d.moves[d.done].crate.SetPosition(d.moves[d.done].from)
	}
	if in.Pressed(input.MouseForward) && d.done < len(d.moves) {
		d.moves[d.done].crate.SetPosition(d.moves[d.done].to)
		d.done++
	}

	if wheel := in.Wheel().Y; wheel != 0 {
		d.eye.SetZoom(min(max(d.eye.Zoom()*(1-0.1*wheel), 0.5), 3))
	}

	switch {
	case d.held != nil:
		d.window.SetCursor(d.carry)
	case under != nil:
		d.window.SetCursor(platform.CursorHand)
	default:
		d.window.SetCursor(platform.CursorArrow)
	}
}

// crateAt is the crate under p in the world, the one drawn last on top.
func (d *Drag) crateAt(p geom.Vec2) *twod.Sprite {
	for i := len(d.crates) - 1; i >= 0; i-- {
		c := d.crates[i]
		at, size := c.Position(), c.Size()
		if (geom.Rect{X: at.X - size.X/2, Y: at.Y - size.Y/2, W: size.X, H: size.Y}).Contains(p) {
			return c
		}
	}
	return nil
}
