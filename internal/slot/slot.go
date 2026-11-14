// Package slot is how engine packages share a value through a World without
// exposing it as a resource: the plugin that owns the value sets it, and the
// engine packages that need it get it. Games hold their values instead.
package slot

import "github.com/lantervale/ember/world"

type slot[T any] struct{ v *T }

// Set makes v the World's T.
func Set[T any](w *world.World, v *T) { world.ResourceOrInit[slot[T]](w).v = v }

// Get is the World's T, nil without one.
func Get[T any](w *world.World) *T {
	if s, ok := world.TryResource[slot[T]](w); ok {
		return s.v
	}
	return nil
}
