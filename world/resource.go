package world

import (
	"fmt"
	"iter"
	"reflect"
	"sort"
)

// SetResource stores a singleton of type T in the World. Resources hold
// state that belongs to no entity: time, input, the active camera, assets.
func SetResource[T any](w *World, v *T) *T {
	w.resources[reflect.TypeFor[T]()] = v
	return v
}

// InsertResource stores a resource given as a pointer, keyed by its element type.
func (w *World) InsertResource(ptr any) {
	t := reflect.TypeOf(ptr)
	if t == nil || t.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("world: resource %T must be a pointer", ptr))
	}
	w.resources[t.Elem()] = ptr
}

// ResourceEntry describes one stored resource.
type ResourceEntry struct {
	Type  reflect.Type
	Value any
}

// Resources lists every resource sorted by type name, for inspectors.
func (w *World) Resources() []ResourceEntry {
	out := make([]ResourceEntry, 0, len(w.resources))
	for t, v := range w.resources {
		out = append(out, ResourceEntry{t, v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type.String() < out[j].Type.String() })
	return out
}

// Resource returns the resource of type T. A missing resource is a wiring
// bug, so it panics with a message naming the type.
func Resource[T any](w *World) *T {
	if v, ok := TryResource[T](w); ok {
		return v
	}
	panic(fmt.Sprintf("world: resource %v not found (is the plugin providing it installed?)", reflect.TypeFor[T]()))
}

func TryResource[T any](w *World) (*T, bool) {
	v, ok := w.resources[reflect.TypeFor[T]()]
	if !ok {
		return nil, false
	}
	p, ok := v.(*T)
	return p, ok
}

// ResourceOrInit returns the resource T, creating a zero value when absent.
func ResourceOrInit[T any](w *World) *T {
	if v, ok := TryResource[T](w); ok {
		return v
	}
	return SetResource(w, new(T))
}

type eventStore interface {
	endFrame(holdForStep bool)
	stepStarted()
}

// eventQueue holds the events of one type: from base, those of the frame
// before frameStart and of this frame after it. stepped is where the queue
// stood when the last fixed step began: every FixedUpdate reader has had
// the events before it.
type eventQueue[T any] struct {
	events     []T
	base       uint64
	frameStart uint64
	stepped    uint64
}

func (q *eventQueue[T]) next() uint64 { return q.base + uint64(len(q.events)) }

func (q *eventQueue[T]) stepStarted() { q.stepped = q.next() }

// endFrame drops the events of the frame before, keeping, when holdForStep,
// those no fixed step has begun after yet.
func (q *eventQueue[T]) endFrame(holdForStep bool) {
	limit := q.frameStart
	if holdForStep {
		limit = max(min(limit, q.stepped), q.base)
	}
	q.events = append(q.events[:0], q.events[limit-q.base:]...)
	q.base = limit
	q.frameStart = q.next()
}

func queueOf[T any](w *World) *eventQueue[T] {
	q, ok := TryResource[eventQueue[T]](w)
	if !ok {
		q = SetResource(w, &eventQueue[T]{})
		w.events = append(w.events, q)
	}
	return q
}

// Emit publishes an event. Events live for two frames so that every reader
// sees each event exactly once regardless of system order, and, while the
// World has FixedUpdate systems, until a fixed step has begun after them,
// so a fixed-step reader sees them however fast the frames run.
func Emit[T any](w *World, ev T) {
	q := queueOf[T](w)
	q.events = append(q.events, ev)
}

// EndFrame ages event queues. Host loops call it once per frame.
func (w *World) EndFrame() {
	hold := w.hasPhase(FixedUpdate)
	for _, q := range w.events {
		q.endFrame(hold)
	}
}

func (w *World) hasPhase(p Phase) bool {
	for _, s := range w.systems {
		if phaseOf(s) == p {
			return true
		}
	}
	return false
}

// Reader tracks which events of type T a consumer has already seen.
// Keep one Reader per consuming system.
type Reader[T any] struct {
	cursor uint64
}

// Any reads the events not yet seen by this reader and reports whether
// there was one: "did anything hit the hero since I last looked?".
func (r *Reader[T]) Any(w *World) bool { return r.Count(w) > 0 }

// Count reads the events not yet seen by this reader and says how many
// there were.
func (r *Reader[T]) Count(w *World) int {
	q := queueOf[T](w)
	r.cursor = max(r.cursor, q.base)
	n := int(q.next() - r.cursor) //nolint:gosec // cursor is between base and next: at most len(q.events)
	r.cursor = q.next()
	return n
}

// Read yields events not yet seen by this reader.
func (r *Reader[T]) Read(w *World) iter.Seq[T] {
	return func(yield func(T) bool) {
		q := queueOf[T](w)
		if r.cursor < q.base {
			r.cursor = q.base
		}
		for r.cursor < q.next() {
			ev := q.events[r.cursor-q.base]
			r.cursor++
			if !yield(ev) {
				return
			}
		}
	}
}
