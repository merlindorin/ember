package node_test

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type log []string

func (l *log) add(s string) { *l = append(*l, s) }

type Plain struct {
	*world.BasicEntity
	node.Node
}

func plain(name string) *Plain { p := &Plain{BasicEntity: world.NewBasic()}; p.SetName(name); return p }

type Traced struct {
	*world.BasicEntity
	node.Node
	log *log
}

func traced(name string, l *log) *Traced {
	t := &Traced{BasicEntity: world.NewBasic(), log: l}
	t.SetName(name)
	return t
}

func (t *Traced) EnterTree()             { t.log.add("enter " + t.Name()) }
func (t *Traced) Ready()                 { t.log.add("ready " + t.Name()) }
func (t *Traced) ExitTree()              { t.log.add("exit " + t.Name()) }
func (t *Traced) Process(float32)        { t.log.add("process " + t.Name()) }
func (t *Traced) PhysicsProcess(float32) { t.log.add("physics " + t.Name()) }
func (t *Traced) Hit(_ int, by string)   { t.log.add("hit " + t.Name() + " " + by) }

func newTree() (*world.World, *node.Tree) {
	w := world.NewWorld()
	return w, node.New(w)
}

func TestNamesAreUniqueAmongSiblings(t *testing.T) {
	_, tree := newTree()
	a, b, c := plain("Enemy"), plain("Enemy"), plain("Enemy")
	tree.Root().AddNode(a)
	tree.Root().AddNode(b)
	tree.Root().AddNode(c)
	assert.Equal(t, []string{"Enemy", "Enemy2", "Enemy3"}, names(tree.Root().ChildNodes()))

	anon := &Plain{BasicEntity: world.NewBasic()}
	tree.Root().AddNode(anon)
	assert.Equal(t, "Plain", anon.Name(), "an unnamed node takes its type's name")

	c.SetName("Boss/1")
	assert.Equal(t, "Boss_1", c.Name(), "a slash would split the path")
	c.SetName("Enemy")
	assert.Regexp(t, `^Enemy[0-9]+$`, c.Name(), "a rename into a sibling's name gets a suffix")
	assert.Len(t, names(tree.Root().ChildNodes()), 4)
}

func names(ns []*node.Node) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = n.Name()
	}
	return out
}

func TestPathsResolveRelativeAndAbsolute(t *testing.T) {
	_, tree := newTree()
	world, player, hand, sun := plain("World"), plain("Player"), plain("Hand"), plain("Sun")
	sword := plain("Sword")
	tree.Root().AddNode(world)
	world.AddNode(player)
	world.AddNode(sun)
	player.AddNode(hand)
	hand.AddNode(sword)
	require.NoError(t, player.SetOwner(&world.Node))
	require.NoError(t, hand.SetOwner(&world.Node))
	require.NoError(t, sword.SetOwner(&world.Node))
	require.NoError(t, sun.SetOwner(&world.Node))

	assert.Equal(t, "/root/World/Player/Hand", hand.Path())
	assert.Same(t, &hand.Node, world.Get("Player/Hand"))
	assert.Same(t, &sun.Node, player.Get("../Sun"))
	assert.Same(t, &player.Node, player.Get("."))
	assert.Same(t, &sun.Node, hand.Get("/root/World/Sun"))
	assert.Same(t, &sword.Node, world.Get("Player/Hand/Sword"))
	assert.Nil(t, world.Get("Player/Nope"))

	p, ok := node.Find[*Plain](&world.Node, "Player")
	assert.True(t, ok)
	assert.Same(t, player, p)
	_, ok = node.Find[*Traced](&world.Node, "Player")
	assert.False(t, ok)
}

func TestLifecycleRunsInTreeOrder(t *testing.T) {
	w, tree := newTree()
	var l log
	a, b, c := traced("A", &l), traced("B", &l), traced("C", &l)
	a.AddNode(b)
	a.AddNode(c)
	assert.Empty(t, l, "nothing runs outside the tree")

	tree.Root().AddNode(a)
	assert.Equal(t, log{"enter A", "enter B", "enter C", "ready B", "ready C", "ready A"}, l)
	for _, e := range []world.Identifier{a, b, c} {
		assert.True(t, w.Has(e))
	}

	l = nil
	tree.Root().RemoveNode(a)
	assert.Equal(t, log{"exit C", "exit B", "exit A"}, l)
	assert.False(t, w.Has(b))
	assert.False(t, a.IsInsideTree())

	l = nil
	tree.Root().AddNode(a)
	assert.Equal(t, log{"enter A", "enter B", "enter C"}, l, "ready runs once")
	assert.Same(t, a.BasicEntity, b.Parent(), "the entity hierarchy survives leaving the world")
	assert.True(t, w.Has(b))

	l = nil
	b.RequestReady()
	tree.Root().RemoveNode(a)
	tree.Root().AddNode(a)
	assert.Equal(t, log{"exit C", "exit B", "exit A", "enter A", "enter B", "enter C", "ready B"}, l)
}

type removals struct {
	world.Group[node.Noded]
	n int
}

func (r *removals) Remove(e world.Identifier) { r.n++; r.Group.Remove(e) }

func TestReparentingDuringProcessKeepsTheNodeAlive(t *testing.T) {
	w, tree := newTree()
	seen := &removals{}
	w.AddView(seen)
	var l log
	a, b := plain("A"), plain("B")
	moved := traced("Moved", &l)
	tree.Root().AddNode(a)
	tree.Root().AddNode(b)
	a.AddNode(moved)
	w.AddSystem(world.Func(world.Update, "move", func(world.Frame) { moved.Reparent(&b.Node) }))
	w.Update(0)
	assert.True(t, w.Has(moved))
	assert.Zero(t, seen.n, "moving within the tree never leaves the world")
	assert.Equal(t, "/root/B/Moved", moved.Path())
	assert.Same(t, b.BasicEntity, moved.ParentNode().Entity().GetBasicEntity())
	assert.Same(t, b.BasicEntity, moved.Parent(), "the entity hierarchy follows the tree")

	moved.Reparent(&a.Node)
	assert.Zero(t, seen.n, "nor outside an update")
	assert.Equal(t, "/root/A/Moved", moved.Path())
}

func TestProcessModesAndPause(t *testing.T) {
	w, tree := newTree()
	var l log
	modes := map[string]node.ProcessMode{
		"pausable": node.Pausable,
		"paused":   node.WhenPaused,
		"always":   node.Always,
		"off":      node.Disabled,
	}
	for _, name := range []string{"pausable", "paused", "always", "off"} {
		n := traced(name, &l)
		n.ProcessMode = modes[name]
		tree.Root().AddNode(n)
		child := traced(name+"-child", &l)
		n.AddNode(child)
	}

	l = nil
	w.UpdatePhase(world.Update, 0)
	assert.Equal(t, log{"process pausable", "process pausable-child", "process always", "process always-child"}, l)

	l = nil
	tree.Paused = true
	w.UpdatePhase(world.Update, 0)
	assert.Equal(t, log{"process paused", "process paused-child", "process always", "process always-child"}, l)

	l = nil
	w.UpdatePhase(world.FixedUpdate, 0)
	assert.Equal(t, log{"physics paused", "physics paused-child", "physics always", "physics always-child"}, l)
}

func TestGroups(t *testing.T) {
	_, tree := newTree()
	var l log
	wolf, bear, rock := traced("Wolf", &l), traced("Bear", &l), traced("Rock", &l)
	wolf.AddToGroup("enemies")
	bear.AddToGroup("enemies")
	tree.Root().AddNode(rock)
	tree.Root().AddNode(bear)
	rock.AddNode(wolf)
	assert.True(t, wolf.IsInGroup("enemies"))
	assert.Equal(t, []string{"Wolf", "Bear"}, names(tree.Group("enemies")), "tree order")

	l = nil
	tree.CallGroup("enemies", "Hit", 3, "sword")
	assert.Equal(t, log{"hit Wolf sword", "hit Bear sword"}, l)

	bear.RemoveFromGroup("enemies")
	tree.Root().RemoveNode(rock)
	assert.Empty(t, tree.Group("enemies"), "only nodes in the tree are listed")
	assert.True(t, wolf.IsInGroup("enemies"))
}

func TestOwnerMustBeAnAncestorAndIsClearedOnRemoval(t *testing.T) {
	_, tree := newTree()
	level, room, lamp, other, bulb := plain("Level"), plain("Room"), plain("Lamp"), plain("Other"), plain("Bulb")
	tree.Root().AddNode(level)
	tree.Root().AddNode(other)
	level.AddNode(room)
	room.AddNode(lamp)
	lamp.AddNode(bulb)
	require.NoError(t, bulb.SetOwner(&level.Node))
	assert.Error(t, lamp.SetOwner(&other.Node))
	require.NoError(t, room.SetOwner(&level.Node))
	require.NoError(t, lamp.SetOwner(&room.Node))

	level.RemoveNode(room)
	assert.Nil(t, room.Owner(), "the removed root leaves its owner's scene")
	assert.Same(t, &room.Node, lamp.Owner(), "nodes keep an owner that left with them")
	assert.Nil(t, bulb.Owner(), "and lose one that stayed behind")
}

func TestQueueFreeRemovesAtTheEndOfTheFrame(t *testing.T) {
	w, tree := newTree()
	var l log
	n := traced("N", &l)
	tree.Root().AddNode(n)
	w.AddSystem(world.Func(world.Update, "free", func(world.Frame) {
		n.QueueFree()
		assert.True(t, n.IsInsideTree())
	}))
	l = nil
	w.Update(0)
	assert.Contains(t, l, "exit N")
	assert.False(t, w.Has(n))
	assert.Empty(t, tree.Root().ChildNodes())
}

type Button struct {
	*world.BasicEntity
	node.Node
	Pressed     node.Signal[struct{}]
	TextChanged node.Signal[string]
	Renamed     node.Signal[string] `signal:"name_changed"`
}

type Door struct {
	*world.BasicEntity
	node.Node
	log *log
}

func (d *Door) OnPressed()           { d.log.add("open") }
func (d *Door) OnText(s string)      { d.log.add("text " + s) }
func (d *Door) OnNumber(int)         {}
func (d *Door) OnTwo(string, string) {}

func TestSignals(t *testing.T) {
	w, tree := newTree()
	var l log
	button := &Button{BasicEntity: world.NewBasic()}
	door := &Door{BasicEntity: world.NewBasic(), log: &l}
	tree.Root().AddNode(button)
	tree.Root().AddNode(door)

	disconnect := button.TextChanged.Connect(func(s string) { l.add("closure " + s) }, 0)
	button.TextChanged.Connect(func(s string) { l.add("once " + s) }, node.OneShot)
	button.TextChanged.Emit("a")
	button.TextChanged.Emit("b")
	disconnect()
	button.TextChanged.Emit("c")
	assert.Equal(t, log{"closure a", "once a", "closure b"}, l)

	l = nil
	require.NoError(t, node.ConnectMethod(button, "pressed", door, "OnPressed", node.Deferred))
	require.NoError(t, node.ConnectMethod(button, "text_changed", door, "OnText", 0))
	require.NoError(t, node.ConnectMethod(button, "name_changed", door, "OnText", 0))
	w.AddSystem(world.Func(world.Update, "press", func(world.Frame) {
		button.Pressed.Emit(struct{}{})
		assert.Empty(t, l, "deferred calls wait for the end of the frame")
	}))
	w.Update(0)
	assert.Equal(t, log{"open"}, l)
	button.Renamed.Emit("x")
	assert.Equal(t, log{"open", "text x"}, l)

	for _, c := range []struct{ signal, method, err string }{
		{"pressed", "Nope", "no method Nope"},
		{"nope", "OnPressed", "no signal nope"},
		{"text_changed", "OnNumber", "string"},
		{"text_changed", "OnTwo", "string"},
		{"text_changed", "OnPressed", "string"},
	} {
		err := node.ConnectMethod(button, c.signal, door, c.method, 0)
		if assert.Error(t, err, c) {
			assert.True(t, strings.Contains(err.Error(), c.err), err.Error())
		}
	}
	assert.Equal(t, []string{"pressed", "text_changed", "name_changed"}, node.SignalNames(button))
}

type Prepared struct {
	*world.BasicEntity
	node.Node
	inWorld bool
}

func (p *Prepared) EnterTree() { p.inWorld = p.Tree().World().Has(p) }

func TestEnterTreeRunsBeforeTheEntityJoinsTheWorld(t *testing.T) {
	w, tree := newTree()
	p := &Prepared{BasicEntity: world.NewBasic()}
	tree.Root().AddNode(p)
	assert.False(t, p.inWorld, "EnterTree prepares components before systems see them")
	assert.True(t, w.Has(p))
}

func TestReparentKeepsAnOwnerThatIsStillAnAncestor(t *testing.T) {
	_, tree := newTree()
	level, room, hall := plain("Level"), plain("Room"), plain("Hall")
	lamp, bulb, other := plain("Lamp"), plain("Bulb"), plain("Other")
	tree.Root().AddNode(level)
	tree.Root().AddNode(other)
	level.AddNode(room)
	level.AddNode(hall)
	room.AddNode(lamp)
	lamp.AddNode(bulb)
	for _, n := range []*Plain{room, hall, lamp, bulb} {
		require.NoError(t, n.SetOwner(&level.Node))
	}

	lamp.Reparent(&hall.Node)
	assert.Same(t, &level.Node, lamp.Owner(), "the level is still above the lamp")
	assert.Same(t, &level.Node, bulb.Owner())

	lamp.Reparent(&other.Node)
	assert.Nil(t, lamp.Owner(), "moved out of the level, the lamp leaves its scene")
	assert.Nil(t, bulb.Owner())
}

func TestTheNameIndexFollowsRenamesAndRemovals(t *testing.T) {
	_, tree := newTree()
	a, b := plain("A"), plain("A")
	tree.Root().AddNode(a)
	tree.Root().AddNode(b)
	assert.Same(t, &b.Node, tree.Root().Get("A2"))
	b.SetName("B")
	assert.Same(t, &b.Node, tree.Root().Get("B"))
	assert.Nil(t, tree.Root().Get("A2"))
	tree.Root().RemoveNode(a)
	assert.Nil(t, tree.Root().Get("A"))
	c := plain("A")
	tree.Root().AddNode(c)
	assert.Equal(t, "A", c.Name(), "a freed name is free again")
}

func BenchmarkAddingNamesakes(b *testing.B) {
	for range b.N {
		root := plain("Root")
		for range 10000 {
			root.AddNode(plain("Sprite"))
		}
	}
}

type quietBase struct {
	pressed node.Signal[struct{}] `signal:"pressed"`
	hidden  node.Signal[int]      //nolint:unused // an untagged unexported field the test proves is no signal
}

type quiet struct {
	*world.BasicEntity
	node.Node
	quietBase
	rang int
}

func (q *quiet) Ring() { q.rang++ }

func TestAnUnexportedTaggedFieldIsASignal(t *testing.T) {
	a, b := &quiet{BasicEntity: world.NewBasic()}, &quiet{BasicEntity: world.NewBasic()}
	if got := node.SignalNames(a); len(got) != 1 || got[0] != "pressed" {
		t.Fatalf("only the tagged one, through an unexported embedded type: %v", got)
	}
	if err := node.ConnectMethod(a, "pressed", b, "Ring", 0); err != nil {
		t.Fatal(err)
	}
	a.pressed.Emit(struct{}{})
	if b.rang != 1 {
		t.Fatal("it connects and emits like an exported one")
	}
}
