//go:build chrome

package chrome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThePagesRunAtTwoPixelsAPointUnlessToldOtherwise(t *testing.T) {
	env := pageEnv([]string{"HOME=/h"})
	assert.Contains(t, env, "EMBER_HIDDEN=1")
	assert.Contains(t, env, "EMBER_SCALE=2", "the thresholds are 2x's, whatever the display")
	env = pageEnv([]string{"EMBER_SCALE=1"})
	assert.Contains(t, env, "EMBER_SCALE=1")
	assert.NotContains(t, env, "EMBER_SCALE=2", "a scale set is kept")
}
