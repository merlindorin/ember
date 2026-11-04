// Package world is the Entity Component System at the centre of ember. The
// World holds entities, systems, views, resources and events, and runs them:
// phases, a fixed step, the clock, plugins, presenters and the runner.
//
// The fixed step accumulates the frames' time and runs FixedUpdate once per
// whole step it holds: RunFrames(n) runs n steps, and a real frame runs
// none, one or several. FixedUpdate runs before Update in each frame, so
// what a system in Update writes for the steps (a player's intent) reaches
// them a frame later; sample input in PreUpdate to use it the same frame.
//
// An entity joins a system's group when it implements the group's
// interface. A component field named like one of its methods (Move,
// Position) hides that method and leaves the entity out without a word;
// World.Explain says which groups an entity is in, and what it lacks for
// the others.
package world

import (
	"iter"
	"reflect"
	"slices"
	"sort"
	"time"
	"unsafe"
)

// World holds the entities, systems, views, resources and events of a game,
// and runs them.
//
// Entities are routed to systems (and views) by the interfaces they
// implement. The World remembers every entity it has seen, so systems added
// late still receive existing entities, and removals requested while a
// system is updating are deferred until that system returns.
type World struct {
	systems    systems
	collectors []*collector
	explicit   map[Collector]bool

	entities map[uint64]Identifier
	// order keeps insertion order with 0 as a tombstone, so removal is O(1);
	// slot maps an ID to its index and dead counts tombstones.
	order []uint64
	slot  map[uint64]int
	dead  int

	resources map[reflect.Type]any
	events    []eventStore

	updating      int
	pendingRemove []Identifier
	// pendingRegroup are entities to offer to the groups again once the
	// running system returns.
	pendingRegroup []Identifier
	started        bool

	stats map[System]*SystemStat

	plugins    map[reflect.Type]bool
	presenters []Presenter
	runner     Runner
	time       *Time
	frameStats *Stats
	quit       bool
}

type collector struct {
	target  Collector
	in, ex  []reflect.Type
	matcher Matcher
	// name is the system field a group was found in, for Explain.
	name string
}

// SystemStat is the smoothed cost of one system, for profilers and
// overlays, and Spent what it cost in the last frame, all its runs added
// (a FixedUpdate system may run several times a frame), for finding what
// a slow frame spent.
type SystemStat struct {
	Name  string
	Phase Phase
	Time  time.Duration
	Spent time.Duration
}

// New returns a World with its clock, frame Stats and the headless runner, ticking at 60 fixed steps a second.
func New() *World {
	w := &World{
		explicit:  map[Collector]bool{},
		entities:  map[uint64]Identifier{},
		resources: map[reflect.Type]any{},
		stats:     map[System]*SystemStat{},
		plugins:   map[reflect.Type]bool{},
		runner:    Headless,
	}
	w.time = &Time{FixedDelta: 1.0 / 60, Scale: 1, MaxDelta: 0.25}
	w.frameStats = &Stats{world: w}
	return w
}

// NewWorld is New.
func NewWorld() *World { return New() }

// AddSystem adds the given System to the World, automatically sorting all systems by priority.
// If the system implements the Initializer interface, its New method is called with the World.
// If it embeds a Group (or otherwise implements Matcher) it starts receiving
// matching entities, including those added before it, and so does every
// other Group field it has.
func (w *World) AddSystem(system System) *World {
	if initializer, ok := system.(Initializer); ok {
		initializer.New(w)
	}

	w.systems = append(w.systems, system)
	sort.Stable(w.systems)
	w.stats[system] = &SystemStat{Name: nameOf(system), Phase: phaseOf(system)}

	_, handles := system.(Collector)
	if c, ok := system.(Collector); ok && !w.explicit[c] {
		if m, matches := system.(Matcher); matches {
			w.addCollector(&collector{target: c, matcher: m})
		}
	}
	for name, g := range groupFields(system, handles) {
		w.addCollector(&collector{target: g, matcher: g, name: name})
	}
	return w
}

// fieldGroup is a Group found in a system's field.
type fieldGroup interface {
	Collector
	Matcher
	wanter
}

// groupFields are the Groups in system's fields, named or embedded,
// exported or not, so a system collects several kinds of entities. An
// embedded group is left out when the system is itself a Collector: its
// own methods fill it.
func groupFields(system System, handles bool) iter.Seq2[string, fieldGroup] {
	return func(yield func(string, fieldGroup) bool) {
		v := reflect.ValueOf(system)
		if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
			return
		}
		v = v.Elem()
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			if f.Anonymous && handles {
				continue
			}
			field := v.Field(i)
			var p any
			switch {
			case f.Type.Kind() == reflect.Struct:
				p = reflect.NewAt(f.Type, unsafe.Pointer(field.UnsafeAddr())).Interface()
			case f.Type.Kind() == reflect.Pointer && f.Type.Elem().Kind() == reflect.Struct:
				ptr := reflect.NewAt(f.Type, unsafe.Pointer(field.UnsafeAddr())).Elem()
				if ptr.IsNil() {
					continue
				}
				p = ptr.Interface()
			default:
				continue
			}
			if g, ok := p.(fieldGroup); ok {
				if !yield(t.Name()+"."+f.Name, g) {
					return
				}
			}
		}
	}
}

// AddSystems adds several systems at once.
func (w *World) AddSystems(systems ...System) *World {
	for _, s := range systems {
		w.AddSystem(s)
	}
	return w
}

// AddSystemInterface adds a system to the world with automatic entity filtering.
// Entities that implement the 'in' interface(s) will be automatically added to the system,
// while entities that implement the 'ex' interface(s) will be excluded.
// Both 'in' and 'ex' can be a single interface or a slice of interfaces.
// The interfaces must be passed as pointers (e.g., (*MyInterface)(nil)).
func (w *World) AddSystemInterface(sys SystemAddByInterfacer, in interface{}, ex interface{}) *World {
	w.explicit[sys] = true
	w.AddSystem(sys)
	w.addCollector(&collector{target: sys, in: interfaceTypes(in), ex: interfaceTypes(ex)})
	return w
}

// AddView registers a collector that is kept in sync with the World but is
// never updated. Renderers, network replicators and debug tools use views to
// observe entities without participating in the simulation.
func (w *World) AddView(view interface {
	Collector
	Matcher
}) *World {
	w.addCollector(&collector{target: view, matcher: view})
	return w
}

// AddViewInterface is AddView with explicit interface filters.
func (w *World) AddViewInterface(view Collector, in interface{}, ex interface{}) *World {
	w.addCollector(&collector{target: view, in: interfaceTypes(in), ex: interfaceTypes(ex)})
	return w
}

func (w *World) addCollector(c *collector) {
	w.collectors = append(w.collectors, c)
	for _, id := range w.order {
		if id != 0 {
			c.offer(w.entities[id])
		}
	}
}

func (c *collector) offer(e Identifier) {
	if c.matcher != nil {
		if c.matcher.Matches(e) {
			c.target.AddByInterface(e)
		}
		return
	}
	t := reflect.TypeOf(e)
	for _, x := range c.ex {
		if t.Implements(x) {
			return
		}
	}
	for _, i := range c.in {
		if t.Implements(i) {
			c.target.AddByInterface(e)
			return
		}
	}
}

func interfaceTypes(v interface{}) []reflect.Type {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		list = []interface{}{v}
	}
	out := make([]reflect.Type, 0, len(list))
	for _, x := range list {
		out = append(out, reflect.TypeOf(x).Elem())
	}
	return out
}

func (w *World) AddEntities(entities ...Identifier) *World {
	for _, entity := range entities {
		w.AddEntity(entity)
	}

	return w
}

// AddEntity adds the given entity to all applicable systems and views.
// Adding an entity twice is a no-op, except that it cancels a removal
// deferred during the current update.
func (w *World) AddEntity(e Identifier) *World {
	if b, ok := e.(BasicFace); ok {
		b.GetBasicEntity().world = w
		defer w.addChildren(b.GetBasicEntity())
	}
	if _, ok := w.entities[e.ID()]; ok {
		w.pendingRemove = slices.DeleteFunc(w.pendingRemove, func(p Identifier) bool { return p.ID() == e.ID() })
		return w
	}
	w.entities[e.ID()] = e
	if w.slot == nil {
		w.slot = map[uint64]int{}
	}
	w.slot[e.ID()] = len(w.order)
	w.order = append(w.order, e.ID())
	for _, c := range w.collectors {
		c.offer(e)
	}
	return w
}

// addChildren adds the children appended as whole entities, so adding a
// parent adds its subtree as removing it removes it.
func (w *World) addChildren(b *BasicEntity) {
	for _, c := range b.children {
		if c.self != nil {
			w.AddEntity(c.self)
		}
	}
}

// Entity returns the entity registered under id.
func (w *World) Entity(id uint64) (Identifier, bool) {
	e, ok := w.entities[id]
	return e, ok
}

// Has reports whether the entity is part of the World.
func (w *World) Has(e Identifier) bool {
	_, ok := w.entities[e.ID()]
	return ok
}

// Entities returns every entity in insertion order. Tools and inspectors
// use it; systems should rely on Groups instead.
func (w *World) Entities() []Identifier {
	out := make([]Identifier, 0, len(w.entities))
	for _, id := range w.order {
		if id != 0 {
			out = append(out, w.entities[id])
		}
	}
	return out
}

// Len returns the number of entities in the World.
func (w *World) Len() int { return len(w.entities) }

// Systems returns a slice of all systems currently registered in the World,
// sorted by their priority.
func (w *World) Systems() []System {
	return w.systems
}

// Time is the World's clock.
func (w *World) Time() *Time { return w.time }

// Stats are the World's frame metrics.
func (w *World) Stats() *Stats { return w.frameStats }

// Timings lists every system's smoothed cost, in run order.
func (w *World) Timings() []*SystemStat {
	out := make([]*SystemStat, 0, len(w.systems))
	for _, s := range w.systems {
		out = append(out, w.stats[s])
	}
	return out
}

// Update runs every phase once. Startup systems only run on the first call.
// The dt parameter represents the time elapsed since the last update in seconds.
func (w *World) Update(dt float32) {
	for p := Startup; p < phaseCount; p++ {
		w.UpdatePhase(p, dt)
	}
}

// UpdatePhase runs the systems of a single phase, letting a host loop run
// FixedUpdate at its own rate. Startup is only ever run once.
func (w *World) UpdatePhase(p Phase, dt float32) {
	if p == Startup {
		if w.started {
			return
		}
		w.started = true
	}
	for _, system := range w.systems {
		if phaseOf(system) != p {
			continue
		}
		begin := time.Now()
		w.updating++
		system.Update(Frame{DT: dt, Elapsed: w.time.Elapsed, FixedFrame: w.time.FixedFrame, Time: w.time, World: w})
		w.updating--
		if st := w.stats[system]; st != nil {
			d := time.Since(begin)
			st.Time, st.Spent = smooth(st.Time, d), st.Spent+d
		}
		w.flush()
	}
}

// RemoveEntity removes the given entity, and its descendants, from all
// systems and views. During an update the removal is deferred until the
// running system returns, so systems may remove entities while iterating.
func (w *World) RemoveEntity(e Identifier) {
	if w.updating > 0 {
		w.pendingRemove = append(w.pendingRemove, e)
		return
	}
	w.remove(e)
}

// TreeMember is an entity that also belongs to a tree of its own, such as
// a scene node: when the World removes it, it leaves that tree's parent
// too.
type TreeMember interface{ LeaveTree() }

// remove takes e and its descendants out of the World. Only e leaves its
// parent: the subtree stays whole, so it can be added again.
func (w *World) remove(e Identifier) {
	if m, ok := e.(TreeMember); ok && w.Has(e) {
		m.LeaveTree()
	}
	if b, ok := e.(BasicFace); ok {
		if basic := b.GetBasicEntity(); basic.parent != nil && w.Has(e) {
			basic.parent.RemoveChild(basic)
		}
	}
	w.removeTree(e)
}

func (w *World) removeTree(e Identifier) {
	registered, ok := w.entities[e.ID()]
	if !ok {
		return
	}
	if b, isBasic := registered.(BasicFace); isBasic {
		b.GetBasicEntity().world = nil
		for _, child := range b.GetBasicEntity().children {
			w.removeTree(w.lookup(child))
		}
	}
	for _, sys := range w.systems {
		if r, removes := sys.(interface{ Remove(Identifier) }); removes {
			r.Remove(registered)
		}
	}
	for _, c := range w.collectors {
		if _, isSystem := c.target.(System); !isSystem {
			c.target.Remove(registered)
		}
	}
	delete(w.entities, e.ID())
	w.order[w.slot[e.ID()]] = 0
	delete(w.slot, e.ID())
	w.dead++
	if w.dead > 64 && w.dead*2 > len(w.order) {
		w.compact()
	}
}

// compact drops tombstones from order, amortized over the removals that
// created them.
func (w *World) compact() {
	live := w.order[:0]
	for _, id := range w.order {
		if id != 0 {
			w.slot[id] = len(live)
			live = append(live, id)
		}
	}
	clear(w.order[len(live):])
	w.order, w.dead = live, 0
}

func (w *World) lookup(b *BasicEntity) Identifier {
	if e, ok := w.entities[b.ID()]; ok {
		return e
	}
	return b
}

func (w *World) flush() {
	for len(w.pendingRemove) > 0 || len(w.pendingRegroup) > 0 {
		p := w.pendingRemove
		w.pendingRemove = nil
		for _, e := range p {
			w.remove(e)
		}
		p = w.pendingRegroup
		w.pendingRegroup = nil
		for _, e := range p {
			w.regroup(e)
		}
	}
}

// Regroup offers e, an entity already in a World, to that World's groups
// again, after a change to it that makes it newly match a group or no
// longer match one: a component set or cleared, an option applied to a
// live entity. A group whose Matches now takes it gets it; one it no
// longer matches lets it go; groups by interface stay as they are, since
// e's type can't change. While a system is updating, it waits until that
// system returns, as removals do, so no group changes under a loop. It
// does nothing for an entity in no World.
func Regroup(e Identifier) {
	b, ok := e.(BasicFace)
	if !ok {
		return
	}
	w := b.GetBasicEntity().world
	if w == nil || !w.Has(e) {
		return
	}
	if w.updating > 0 {
		w.pendingRegroup = append(w.pendingRegroup, e)
		return
	}
	w.regroup(e)
}

func (w *World) regroup(e Identifier) {
	e, ok := w.entities[e.ID()]
	if !ok {
		return
	}
	for _, c := range w.collectors {
		if c.matcher == nil {
			continue
		}
		if c.matcher.Matches(e) {
			c.target.AddByInterface(e)
		} else if _, isSystem := c.target.(System); !isSystem {
			c.target.Remove(e)
		}
	}
}

// SortSystems sorts all systems in the World by their priority.
// This is automatically called when adding systems, but can be
// manually invoked if system priorities are changed at runtime.
func (w *World) SortSystems() {
	sort.Stable(w.systems)
}

func smooth(prev, sample time.Duration) time.Duration {
	if prev == 0 {
		return sample
	}
	return time.Duration(float64(prev)*0.9 + float64(sample)*0.1)
}
