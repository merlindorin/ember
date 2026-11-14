//go:build !js

package ember

import (
	"github.com/lantervale/ember/platform/desktop"
	"github.com/lantervale/ember/world"
)

// runnerPlugin runs the World in native windows.
func runnerPlugin(crashLog string) world.Plugin { return desktop.Plugin{CrashLog: crashLog} }
