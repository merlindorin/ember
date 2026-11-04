package wgpu

import (
	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/world"
)

// Shared is what the desktop runner shares with the renderer in a World:
// the device every window draws with, and what each window draws into
// this frame, keyed by its *platform.Window.
type Shared struct {
	Device  *Device
	Adapter AdapterInfo
	targets map[any]*Texture
}

// Share records d as w's device.
func Share(w *world.World, d *Device, info AdapterInfo) *Shared {
	s := &Shared{Device: d, Adapter: info, targets: map[any]*Texture{}}
	slot.Set(w, s)
	return s
}

// SharedOf is w's shared device, nil when w doesn't draw through WebGPU.
func SharedOf(w *world.World) *Shared { return slot.Get[Shared](w) }

// SetTarget is what win draws into this frame, nil when it draws nothing.
func (s *Shared) SetTarget(win any, t *Texture) {
	if t == nil {
		delete(s.targets, win)
		return
	}
	s.targets[win] = t
}

// Target is what win draws into this frame.
func (s *Shared) Target(win any) *Texture { return s.targets[win] }
