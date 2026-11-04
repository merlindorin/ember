package node_test

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Clock struct {
	WaitTime  float32 `node:"prop"`
	Autostart bool    `node:"prop"`
	left      float32 //nolint:unused // an untagged unexported field the test proves is no property
}

type TimerLike struct {
	*world.BasicEntity
	node.Node
	Clock
	OneShot bool `node:"prop"`
	Timeout node.Signal[struct{}]
}

type Lamp struct {
	*world.BasicEntity
	node.Node
}

type Spawner struct {
	TimerLike
	Wolves int
}

func TestRegisteredTypesAndBehaviours(t *testing.T) {
	node.Register(node.Type{
		Name: "TestTimer",
		New:  func() node.Noded { return &TimerLike{BasicEntity: world.NewBasic()} },
	})
	node.Register(node.Type{
		Name: "test.Spawner", Base: "TestTimer",
		New: func() node.Noded { return &Spawner{TimerLike: TimerLike{BasicEntity: world.NewBasic()}} },
	})
	assert.Panics(t, func() { node.Register(node.Type{Name: "TestTimer", New: func() node.Noded { return nil }}) })
	node.Register(node.Type{New: func() node.Noded { return &Lamp{} }})
	_, ok := node.Lookup("node_test.Lamp")
	assert.True(t, ok, "a type without a name is named by its package and type")

	e, err := node.Create("TestTimer", "")
	require.NoError(t, err)
	assert.IsType(t, &TimerLike{}, e)
	e, err = node.Create("TestTimer", "test.Spawner")
	require.NoError(t, err)
	assert.IsType(t, &Spawner{}, e)

	_, err = node.Create("Nope", "")
	assert.ErrorContains(t, err, "Nope")
	_, err = node.Create("node3d.Node", "test.Spawner")
	assert.ErrorContains(t, err, "TestTimer", "a behaviour only extends its own class")
	_, err = node.Create("TestTimer", "test.Nope")
	assert.ErrorContains(t, err, "test.Nope")
}

func TestPropertiesByName(t *testing.T) {
	s := &Spawner{TimerLike: TimerLike{BasicEntity: world.NewBasic()}}
	assert.Equal(t, []string{"auto_translate_mode", "wait_time", "autostart", "one_shot"}, node.Properties(s))

	require.NoError(t, node.SetProperty(s, "wait_time", json.RawMessage(`30`)))
	require.NoError(t, node.SetProperty(s, "one_shot", json.RawMessage(`true`)))
	assert.Equal(t, float32(30), s.WaitTime)
	assert.True(t, s.OneShot)
	v, err := node.GetProperty(s, "wait_time")
	require.NoError(t, err)
	assert.JSONEq(t, `30`, string(v))

	assert.ErrorContains(t, node.SetProperty(s, "speed", json.RawMessage(`1`)), "speed")
	assert.ErrorContains(t, node.SetProperty(s, "wait_time", json.RawMessage(`"soon"`)), "wait_time")
	_, err = node.GetProperty(s, "speed")
	assert.ErrorContains(t, err, "speed")
}

func TestClassOfAndPersistedConnections(t *testing.T) {
	s, err := node.Create("TestTimer", "test.Spawner")
	require.NoError(t, err)
	class, behaviour, ok := node.ClassOf(s)
	assert.True(t, ok)
	assert.Equal(t, "TestTimer", class)
	assert.Equal(t, "test.Spawner", behaviour)
	_, _, ok = node.ClassOf(&Plain{BasicEntity: world.NewBasic()})
	assert.False(t, ok)

	var l log
	door := &Door{BasicEntity: world.NewBasic(), log: &l}
	timer, ok := s.(*Spawner)
	require.True(t, ok)
	require.NoError(t, node.ConnectMethod(timer, "timeout", door, "OnPressed", node.Persist))
	require.NoError(t, node.ConnectMethod(timer, "timeout", door, "OnPressed", 0))
	timer.Timeout.Connect(func(struct{}) {}, node.Persist)
	assert.Equal(
		t,
		[]node.MethodConnection{{Signal: "timeout", Target: &door.Node, Method: "OnPressed", Flags: node.Persist}},
		node.Connections(timer),
		"only persisted method connections are saved",
	)
}

type shade struct {
	strength float32 `node:"prop"`
}

type Torch struct {
	*world.BasicEntity
	node.Node
	shade
	sortByY bool `node:"prop"`
}

func TestUnexportedFieldsAreProperties(t *testing.T) {
	torch := &Torch{BasicEntity: world.NewBasic()}
	torch.Bind(torch)
	assert.Equal(
		t,
		[]string{"strength", "sort_by_y"},
		node.Properties(torch)[len(node.Properties(torch))-2:],
		"unexported tagged fields, in embedded unexported structs too",
	)
	require.NoError(t, node.SetProperty(torch, "strength", json.RawMessage(`0.5`)))
	require.NoError(t, node.SetProperty(torch, "sort_by_y", json.RawMessage(`true`)))
	assert.Equal(t, float32(0.5), torch.strength)
	assert.True(t, torch.sortByY)
	got, err := node.GetProperty(torch, "sort_by_y")
	require.NoError(t, err)
	assert.JSONEq(t, `true`, string(got))
}
