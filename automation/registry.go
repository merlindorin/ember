// Package automation makes a running ember app drivable from outside: every
// UI element records itself here each frame under a stable ID, apps expose
// named actions, and any live value can be read or written through a
// reflection path. An Automation serves it all over gRPC (with server
// reflection, so grpcurl and Postman discover the API by themselves). Run
// a game that passes ember.Automation with EMBER_AUTOMATION set, then:
//
//	grpcurl -plaintext 127.0.0.1:50551 list
//	grpcurl -plaintext -d '{"prefix":"menu"}' 127.0.0.1:50551 ember.automation.v1.Automation/GetUI
//	grpcurl -plaintext -d '{"id":"menu/play","act":"press"}' 127.0.0.1:50551 ember.automation.v1.Automation/Act
//
// UIs are immediate mode, so manipulation is too: an act is queued for an
// element ID and delivered to the widget the next time it draws, which
// then behaves exactly as if the user had clicked or typed.
package automation

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/merlindorin/ember/geom"
)

// Roles of recorded elements.
const (
	RoleButton    = "button"
	RoleToggle    = "toggle"
	RoleSlider    = "slider"
	RoleTextInput = "text-input"
	RoleSelect    = "selectable"
	RoleMenuItem  = "menu-item"
	RoleBar       = "bar"
	RoleMeter     = "meter"
	RoleLabel     = "label"
	RoleHeading   = "heading"
	RoleTreeItem  = "tree-item"
	RoleCombo     = "combo"
	RoleDialog    = "dialog"
)

// Acts accepted by elements.
const (
	ActPress  = "press"
	ActSet    = "set"
	ActType   = "type"
	ActSubmit = "submit"
	// ActMove moves an element, such as a Movable panel by its title, to
	// the "x,y" its value gives, in window points.
	ActMove = "move"
	// ActResize resizes an element, such as a Resizable panel, to the
	// "w,h" its value gives, in points.
	ActResize = "resize"
	// ActSelect selects one item of a choice: its index, or its name.
	ActSelect = "select"
	// ActContext right-clicks an element (decision 0062).
	ActContext = "context"
	// ActDrag drags a draggable element onto the element whose path its
	// value gives (decision 0062).
	ActDrag = "drag"
)

// Element is one UI element as drawn during a frame.
type Element struct {
	ID      string
	Role    string
	Label   string
	Value   any
	Enabled bool
	Bounds  geom.Rect
	Window  string
	Acts    []string
	Focused bool
}

// Act is a manipulation delivered to a widget.
type Act struct {
	Kind  string
	Value any

	done chan Result
	age  int
}

// Result is the outcome of a queued act.
type Result struct {
	Element Element
	Err     error
}

// ErrNotFound is returned when an act targets an element that is not drawn.
var ErrNotFound = errors.New("automation: element not drawn")

// actTimeout is how many frames an act waits for its element to be drawn.
const actTimeout = 30

// Registry is what widgets record into. While it isn't recording (no
// automation server), recording is free. The zero value is ready to use.
type Registry struct {
	recording bool

	mu        sync.Mutex
	building  []Element
	seen      map[string]int
	published []Element
	frame     uint64
	pending   map[string][]*Act
	taken     map[string]*Act
	actions   map[string]Action
	watchers  []chan struct{}
}

// Recording reports whether widgets record into r: on while the API is
// served, or once SetRecording or the Recording option turned it on.
func (r *Registry) Recording() bool { return r != nil && r.recording }

// SetRecording turns recording on or off, for tests reading what was drawn.
func (r *Registry) SetRecording(on bool) { r.recording = on }

func NewRegistry() *Registry {
	r := &Registry{}
	r.init()
	return r
}

func (r *Registry) init() {
	if r.seen == nil {
		r.seen, r.pending, r.taken, r.actions = map[string]int{}, map[string][]*Act{}, map[string]*Act{}, map[string]Action{}
	}
}

// Take returns the act queued for the element id, if any. Widgets call it
// before drawing, apply the act, then Record their final state.
func (r *Registry) Take(id string) *Act {
	if !r.Recording() {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	q := r.pending[id]
	if len(q) == 0 {
		return nil
	}
	a := q[0]
	r.pending[id] = q[1:]
	r.taken[id] = a
	return a
}

// Record adds an element drawn this frame. Duplicate IDs within a frame get
// a "#n" suffix so every element stays addressable.
func (r *Registry) Record(e Element) {
	if !r.Recording() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	if n := r.seen[e.ID]; n > 0 {
		r.seen[e.ID] = n + 1
		e.ID = fmt.Sprintf("%s#%d", e.ID, n+1)
	} else {
		r.seen[e.ID] = 1
	}
	r.building = append(r.building, e)
	if a, ok := r.taken[e.ID]; ok {
		delete(r.taken, e.ID)
		a.done <- Result{Element: e}
	}
}

// Queue schedules an act for an element and returns a channel receiving
// the element once the act was applied (or an error).
func (r *Registry) Queue(id, kind string, value any) <-chan Result {
	a := &Act{Kind: kind, Value: value, done: make(chan Result, 1)}
	r.mu.Lock()
	r.init()
	r.pending[id] = append(r.pending[id], a)
	r.mu.Unlock()
	return a.done
}

// EndFrame publishes the elements recorded since the last call, fails acts
// whose element never showed up, and wakes watchers. The plugin calls it
// once per frame.
func (r *Registry) EndFrame(frame uint64) {
	r.mu.Lock()
	r.published, r.building = r.building, r.published[:0]
	clear(r.seen)
	r.frame = frame
	for id, a := range r.taken {
		a.done <- Result{Err: fmt.Errorf("%w: %s consumed an act but did not record itself", ErrNotFound, id)}
		delete(r.taken, id)
	}
	for id, q := range r.pending {
		live := q[:0]
		for _, a := range q {
			if a.age++; a.age > actTimeout {
				a.done <- Result{Err: fmt.Errorf("%w: %s", ErrNotFound, id)}
				continue
			}
			live = append(live, a)
		}
		if len(live) == 0 {
			delete(r.pending, id)
		} else {
			r.pending[id] = live
		}
	}
	watchers := r.watchers
	r.mu.Unlock()
	for _, w := range watchers {
		select {
		case w <- struct{}{}:
		default:
		}
	}
}

// Snapshot returns the last published frame, filtered by window title and
// id prefix (empty strings match everything).
func (r *Registry) Snapshot(window, prefix string) (uint64, []Element) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Element, 0, len(r.published))
	for _, e := range r.published {
		if (window == "" || e.Window == window) && strings.HasPrefix(e.ID, prefix) {
			out = append(out, e)
		}
	}
	return r.frame, out
}

// Action is a named intent apps expose, invokable by automation clients.
// Buttons that trigger game logic should call the same action so humans
// and tools share one code path.
type Action struct {
	ID    string
	Label string
	// Enabled reports whether the action can run now (nil means always).
	Enabled func() bool
	Run     func(args map[string]any) (any, error)
}

// Define registers (or replaces) an action.
func (r *Registry) Define(a Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.init()
	r.actions[a.ID] = a
}

// Actions lists actions sorted by ID, optionally filtered by prefix.
func (r *Registry) Actions(prefix string) []Action {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Action, 0, len(r.actions))
	for id, a := range r.actions {
		if strings.HasPrefix(id, prefix) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Invoke runs an action. It must be called on the main loop.
func (r *Registry) Invoke(id string, args map[string]any) (any, error) {
	r.mu.Lock()
	a, ok := r.actions[id]
	r.mu.Unlock()
	switch {
	case !ok:
		return nil, fmt.Errorf("automation: unknown action %q", id)
	case a.Enabled != nil && !a.Enabled():
		return nil, fmt.Errorf("automation: action %q is disabled", id)
	}
	return a.Run(args)
}

// Slug turns a label into an ID segment: "Craft Axe!" → "craft-axe".
// An ImGui-style "label##id" uses the part after ##.
func Slug(label string) string {
	if i := strings.Index(label, "##"); i >= 0 {
		label = label[i+2:]
	}
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(label) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

// Join builds an element ID from scope segments, skipping empty ones.
func Join(parts ...string) string {
	out := parts[:0:0]
	for _, p := range parts {
		if p = strings.Trim(p, "/"); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}
