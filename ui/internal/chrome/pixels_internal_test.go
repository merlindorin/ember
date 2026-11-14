//go:build chrome

package chrome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThePagesRunAtTwoPixelsAPointUnlessToldOtherwise(t *testing.T) {
	env := pageEnv([]string{"HOME=/h"}, "/tmp/run")
	assert.Contains(t, env, "EMBER_HIDDEN=1")
	assert.Contains(t, env, "EMBER_SCALE=2", "the thresholds are 2x's, whatever the display")
	assert.Contains(t, env, "EMBER_CRASH_DIR=/tmp/run", "a page that fails logs in the test's folder")
	env = pageEnv([]string{"EMBER_SCALE=1", "EMBER_CRASH_DIR=/mine"}, "/tmp/run")
	assert.Contains(t, env, "EMBER_SCALE=1")
	assert.NotContains(t, env, "EMBER_SCALE=2", "a scale set is kept")
	assert.Contains(t, env, "EMBER_CRASH_DIR=/mine")
	assert.NotContains(t, env, "EMBER_CRASH_DIR=/tmp/run", "a crash folder set is kept")
}
