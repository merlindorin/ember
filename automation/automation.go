package automation

import (
	"fmt"
	"log"
	"os"

	"github.com/merlindorin/ember/internal/slot"
	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
)

// Automation records a World's widgets and the game's actions and, given an
// address, serves them with Connect over HTTP/2 without TLS, which gRPC and
// Connect clients both speak, with server reflection for grpcurl. It is off
// by default: set Addr or EMBER_AUTOMATION (e.g. "127.0.0.1:50551").
// Games make one, pass it to ember.New with ember.Automation, and Define
// their actions on it.
type Automation struct {
	*Registry
	scenes      *node.Tree
	addr, token string
	// stop shuts the server down, nil when not serving.
	stop     func()
	listened string
}

// Option sets up an Automation.
type Option func(*Automation)

// Addr serves the API on addr (default EMBER_AUTOMATION).
func Addr(addr string) Option { return func(a *Automation) { a.addr = addr } }

// Token requires "Authorization: Bearer <token>" on every call
// (default EMBER_AUTOMATION_TOKEN).
func Token(token string) Option { return func(a *Automation) { a.token = token } }

// Scenes serves tree through SceneService, for clients to inspect and
// change a running game's scene.
func Scenes(tree *node.Tree) Option { return func(a *Automation) { a.scenes = tree } }

// Recording records the widgets without serving them, for tests that read
// what a frame drew.
func Recording() Option { return func(a *Automation) { a.recording = true } }

// New returns an Automation set up by opts.
func New(opts ...Option) *Automation {
	a := &Automation{
		Registry: NewRegistry(),
		addr:     os.Getenv("EMBER_AUTOMATION"),
		token:    os.Getenv("EMBER_AUTOMATION_TOKEN"),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Addr is the address actually listened on (useful with port 0), empty
// when not serving.
func (a *Automation) Addr() string { return a.listened }

// Stop shuts the server down.
func (a *Automation) Stop() {
	if a.stop != nil {
		a.stop()
	}
}

func (a *Automation) Build(w *world.World) {
	slot.Set(w, a.Registry)
	ctl := &controller{world: w, reg: a.Registry, scenes: a.scenes, jobs: make(chan func(), 256)}
	w.AddSystems(&frameSystem{ctl: ctl})
	if a.addr == "" {
		return
	}
	stop, addr, err := serve(a.addr, a.token, ctl)
	if err != nil {
		log.Printf("automation: %v", err)
		return
	}
	a.stop, a.listened = stop, addr
	a.recording = true
	log.Printf("automation: Connect and gRPC API on %s (server reflection enabled)", addr)
}

// controller runs requests on the main loop and tracks time control and
// pending screenshots.
type controller struct {
	world  *world.World
	reg    *Registry
	scenes *node.Tree
	jobs   chan func()

	paused     bool
	controlled bool
	steps      int
	stepDone   []chan uint64
	shots      []*shot
}

type shot struct {
	path   string
	frames int
	done   chan shotResult
}

type shotResult struct {
	png []byte
	err error
}

// frameSystem publishes the UI of the frame that just ended, then runs
// queued requests and time control before the new frame's systems.
type frameSystem struct{ ctl *controller }

func (s *frameSystem) Phase() world.Phase      { return world.PreUpdate }
func (s *frameSystem) Priority() int           { return 1 << 30 }
func (s *frameSystem) Remove(world.Identifier) {}
func (s *frameSystem) Name() string            { return "automation" }

func (s *frameSystem) Update(world.Frame) {
	c := s.ctl
	t := c.world.Time()
	c.reg.EndFrame(t.Frame)
	for {
		select {
		case job := <-c.jobs:
			job()
			continue
		default:
		}
		break
	}
	c.timeControl(t)
	c.collectShots()
}

func (c *controller) timeControl(t *world.Time) {
	switch {
	case c.steps > 0:
		t.Scale = 1
		c.steps--
		if c.steps == 0 {
			c.paused = true
			for _, ch := range c.stepDone {
				ch <- t.Frame
			}
			c.stepDone = nil
		}
		c.controlled = true
	case c.paused:
		t.Scale = 0
		c.controlled = true
	case c.controlled:
		t.Scale = 1
		c.controlled = false
	}
}

func (c *controller) collectShots() {
	live := c.shots[:0]
	for _, s := range c.shots {
		s.frames++
		if data, err := os.ReadFile(s.path); err == nil && len(data) > 0 {
			_ = os.Remove(s.path)
			s.done <- shotResult{png: data}
			continue
		}
		if s.frames > 120 {
			s.done <- shotResult{err: fmt.Errorf("automation: no frame presented for the screenshot (headless app?)")}
			continue
		}
		live = append(live, s)
	}
	c.shots = live
}
