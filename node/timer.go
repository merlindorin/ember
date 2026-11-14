package node

import "github.com/lantervale/ember/world"

func init() { //nolint:gochecknoinits // registers the package's node types on import
	Register(Type{New: func() Noded { return NewEmpty() }})
	Register(Type{New: func() Noded { return NewTimer() }})
}

// Empty is a node with no data of its own: a group, or the base of a
// behaviour.
type Empty struct {
	*world.BasicEntity
	Node
}

func NewEmpty() *Empty { return &Empty{BasicEntity: world.NewBasic()} }

// Timer counts down in Process and emits timeout, again every WaitTime
// unless OneShot.
type Timer struct {
	Empty
	WaitTime  float32 `node:"prop"`
	OneShot   bool    `node:"prop"`
	Autostart bool    `node:"prop"`
	Timeout   Signal[struct{}]

	left    float32
	running bool
}

func NewTimer() *Timer { return &Timer{Empty: *NewEmpty(), WaitTime: 1} }

func (t *Timer) Ready() {
	if t.Autostart {
		t.Start()
	}
}

func (t *Timer) Start()            { t.left, t.running = t.WaitTime, t.WaitTime > 0 }
func (t *Timer) Stop()             { t.left, t.running = 0, false }
func (t *Timer) IsStopped() bool   { return !t.running }
func (t *Timer) TimeLeft() float32 { return t.left }

func (t *Timer) Process(dt float32) {
	if !t.running {
		return
	}
	for t.left -= dt; t.running && t.left <= 0; {
		t.Timeout.Emit(struct{}{})
		if t.OneShot || t.WaitTime <= 0 {
			t.Stop()
		} else {
			t.left += t.WaitTime
		}
	}
}
