//go:build chrome

package chrome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPlatformsThresholdReplacesTheTestsOwn(t *testing.T) {
	m := map[string]float64{"09-button": 0.009}
	assert.Equal(t, 0.009, thresholdFor("09-button", 0.0066, m))
	assert.Equal(t, 0.002, thresholdFor("m0", 0.002, m))
	assert.NotPanics(t, func() { overrides() }, "this system's file, if any, parses")
}
