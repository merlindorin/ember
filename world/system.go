package world

import "reflect"

// A System implements logic for processing entities possessing components of
// the same aspects as the system. A System should iterate over its Entities on
// `Update`, in any way suitable for the current implementation.
//
// By convention, systems provide an Add method for adding entities and their
// associated components to the system; e.g.
//
//	Add(basic *world.BasicEntity, collision *CollisionComponent, space *SpaceComponent)
//
// Most systems do not need to write that plumbing: embed a Group[T] and the
// World routes every entity implementing T to the system automatically.
//
// A system that keeps entities of its own also implements Remove(Identifier)
// to drop them; embedding a Group provides it.
type System interface {
	// Update runs the system once per frame of its phase.
	Update(f Frame)
}

// SystemAddByInterfacer is a system that also implements the AddByInterface method.
type SystemAddByInterfacer interface {
	System

	// AddByInterface allows you to automatically add entities based on the
	// interfaces that the entity implements. It should add the entity passed
	// as o to the system after casting it to the correct interface.
	AddByInterface(o Identifier)
	Remove(e Identifier)
}

// Collector receives entities matching an interface without being updated
// every frame. Systems are collectors; so are Views.
type Collector interface {
	AddByInterface(o Identifier)
	Remove(e Identifier)
}

// Matcher is implemented by collectors that know which entities they want,
// such as Group[T]. The World uses it to route entities with no explicit
// AddSystemInterface call.
type Matcher interface {
	Matches(o Identifier) bool
}

// Prioritizer specifies the priority of systems.
type Prioritizer interface {
	// Priority indicates the order in which Systems should be executed per
	// iteration, higher meaning sooner. The default priority is 0.
	Priority() int
}

type Phase uint8

const (
	Startup     Phase = iota // runs once, before the first frame: LoadAssetsSystem
	PreUpdate                // input, networking receive: NetworkSystem
	FixedUpdate              // deterministic simulation at a fixed rate: PhysicsSystem
	Update                   // gameplay: AISystem
	PostUpdate               // derived state: TransformSystem, CameraSystem
	phaseCount

	// Deprecated: rendering is presentation, not simulation; renderers
	// observe the World through views. Kept so legacy modules compile.
	Render = PostUpdate
)

func (p Phase) String() string {
	phaseNames := [...]string{"Startup", "PreUpdate", "FixedUpdate", "Update", "PostUpdate"}
	if int(p) < len(phaseNames) {
		return phaseNames[p]
	}
	return "Phase(?)"
}

// Phaser specifies the execution phase of systems.
type Phaser interface {
	// Phase returns the execution phase for this system (Startup, PreUpdate,
	// FixedUpdate, Update, PostUpdate). Systems execute in phase order, then by
	// priority within each phase. The default phase is Update.
	Phase() Phase
}

// Initializer provides initialization of systems.
type Initializer interface {
	// New initializes the given System, and may be used to initialize some
	// values beforehand, like storing a reference to the World.
	New(*World)
}

type In interface {
	In() []interface{}
}

type Ex interface {
	Ex() []interface{}
}

func phaseOf(s System) Phase {
	if p, ok := s.(Phaser); ok {
		return p.Phase()
	}
	return Update
}

func priorityOf(s System) int {
	if p, ok := s.(Prioritizer); ok {
		return p.Priority()
	}
	return 0
}

func nameOf(s any) string {
	if n, ok := s.(interface{ Name() string }); ok {
		return n.Name()
	}
	t := reflect.TypeOf(s)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Name() != "" {
		return t.Name()
	}
	return t.String()
}

// systems implements a sortable list of `System`. It is indexed on
// `System.Priority()`.
type systems []System

// Len returns the number of systems in the slice.
// This method is part of the sort.Interface implementation.
func (s systems) Len() int {
	return len(s)
}

// Less reports whether the system at index i should sort before
// the system at index j based on their phase and priority.
// Systems are sorted by phase first (Startup, PreUpdate, FixedUpdate,
// Update, PostUpdate), then by priority within each phase (higher priority
// values sort first).
// This method is part of the sort.Interface implementation.
func (s systems) Less(i, j int) bool {
	phase1, phase2 := phaseOf(s[i]), phaseOf(s[j])
	if phase1 != phase2 {
		return phase1 < phase2
	}
	return priorityOf(s[i]) > priorityOf(s[j])
}

// Swap exchanges the systems at indices i and j.
// This method is part of the sort.Interface implementation.
func (s systems) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// SystemFunc turns a plain function into a System, for logic that does not
// track entities (spawners, timers, UI drawing).
//
//	w.AddSystem(world.Func(world.Update, "spawn-wave", func(dt float32) { ... }))
func Func(phase Phase, name string, fn func(f Frame)) System {
	return &funcSystem{phase: phase, name: name, fn: fn}
}

type funcSystem struct {
	phase Phase
	name  string
	fn    func(f Frame)
}

func (f *funcSystem) Update(fr Frame) { f.fn(fr) }
func (f *funcSystem) Phase() Phase    { return f.phase }
func (f *funcSystem) Name() string    { return f.name }
