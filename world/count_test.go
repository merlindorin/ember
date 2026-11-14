package world_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/world"
)

func TestCountersAddUpInTheOrderFirstCounted(t *testing.T) {
	var s world.Stats
	s.Count("b")
	s.Count("a")
	s.Count("b")
	assert.Equal(t, []*world.Counter{{Name: "b", N: 2}, {Name: "a", N: 1}}, s.Counted())
}
