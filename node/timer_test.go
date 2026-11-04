package node_test

import (
	"testing"

	"github.com/merlindorin/ember/node"
	"github.com/merlindorin/ember/world"
	"github.com/stretchr/testify/assert"
)

func TestTimer(t *testing.T) {
	w := world.NewWorld()
	tree := node.New(w)
	timer := node.NewTimer()
	timer.WaitTime, timer.OneShot = 1, true
	fired := 0
	timer.Timeout.Connect(func(struct{}) { fired++ }, 0)
	tree.Root().AddNode(timer)
	w.Update(2)
	assert.Zero(t, fired, "a timer without autostart waits for Start")
	timer.Start()
	w.Update(0.6)
	assert.InDelta(t, 0.4, timer.TimeLeft(), 1e-5)
	w.Update(0.6)
	w.Update(5)
	assert.Equal(t, 1, fired, "one shot")
	assert.True(t, timer.IsStopped())
}
